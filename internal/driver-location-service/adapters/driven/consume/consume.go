package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"

	bm "ride-hail/internal/driver-location-service/adapters/driven/bm"
	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/ports/driven"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Константы
const (
	rideExchange     = "ride_topic"
	driverTopic      = "driver_topic"
	driverMatchingQ  = "driver_matching"
	rideRequestBind  = "ride.request.*"
	defaultPrefetch  = 50
	offerTTL         = 30 * time.Second
	globalMatchLimit = 120 * time.Second
)

type MatchingConsumer struct {
	cfg    *config.Config
	log    mylogger.Logger
	broker driven.IDriverBroker

	ctx    context.Context
	appCtx context.Context

	mu sync.Mutex
	wg sync.WaitGroup
}

func NewMatchingConsumer(
	ctx context.Context,
	appCtx context.Context,
	cfg *config.Config,
	log mylogger.Logger,
) *MatchingConsumer {
	return &MatchingConsumer{
		ctx:    ctx,
		appCtx: appCtx,
		cfg:    cfg,
		log:    log,
	}
}

func (c *MatchingConsumer) Run() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	l := c.log.Action("driver-matching-run")

	if err := c.initBroker(); err != nil {
		l.Action("mb_connect_failed").Error("rabbitmq connect failed", err)
		return err
	}
	l.Action("mb_connected").Info("rabbitmq connected")

	msgs, err := c.broker.Consume(
		c.appCtx,
		driverMatchingQ,
		rideRequestBind,
		driven.ConsumeOptions{Prefetch: defaultPrefetch, AutoAck: false, QueueDurable: true},
	)
	if err != nil {
		return fmt.Errorf("consume driver_matching: %w", err)
	}

	go c.loop(msgs, c.processRideRequest)

	l.Action("consumer_started").Info("driver-matching consumer started",
		"queue", driverMatchingQ, "binding", rideRequestBind, "prefetch", defaultPrefetch)

	return nil
}

func (c *MatchingConsumer) Stop(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Action("shutdown").Info("graceful shutdown started")

	c.wg.Wait()

	if c.broker != nil {
		if err := c.broker.Close(); err != nil {
			c.log.Action("mb_close_failed").Error("rabbit close failed", err)
			return err
		}
	}

	c.log.Action("shutdown_done").Info("graceful shutdown done")
	return nil
}

func (c *MatchingConsumer) loop(
	dlv <-chan amqp.Delivery,
	processor func(amqp.Delivery) (requeue bool, err error),
) {
	for {
		select {
		case <-c.ctx.Done():
			c.log.Info("stop consumer: context done")
			return
		case m, ok := <-dlv:
			if !ok {
				c.log.Info("stop consumer: channel closed")
				return
			}
			c.wg.Add(1)
			go func(msg amqp.Delivery) {
				defer c.wg.Done()
				requeue, err := processor(msg)
				if err != nil {
					c.log.Error("process error", err)
					_ = msg.Nack(false, requeue)
					return
				}
				_ = msg.Ack(false)
			}(m)
		}
	}
}

func (c *MatchingConsumer) processRideRequest(msg amqp.Delivery) (bool, error) {
	var req dto.RideRequest
	if err := json.Unmarshal(msg.Body, &req); err != nil {
		return false, fmt.Errorf("decode ride.request: %w", err)
	}

	deadline := time.Now().Add(globalMatchLimit)
	if req.MatchTimeoutSec > 0 {
		deadline = time.Now().Add(time.Duration(req.MatchTimeoutSec) * time.Second)
	}

	c.log.Action("matching_start").Info("start matching",
		"ride_id", req.RideID, "type", req.RideType, "pickup", req.Pickup)

	// TODO 1: выбрать кандидатов из БД по статусу AVAILABLE и последним координатам в радиусе
	// TODO 2: разослать офферы по WS водителям [driver_id...]
	// TODO 3: ждать первый accept до offerTTL и общий deadline
	// Заглушка: схитрим и сразу публикуем "accepted" от условного водителя:
	fakeDriverID := "00000000-0000-0000-0000-000000000001"
	if err := c.publishDriverAccepted(req, fakeDriverID, 180); err != nil {
		return true, fmt.Errorf("publish accept: %w", err)
	}

	c.log.Action("matching_done").Info("matching finished (STUB)", "ride_id", req.RideID, "driver_id", fakeDriverID)
	_ = deadline
	return false, nil
}

func (c *MatchingConsumer) publishDriverAccepted(req dto.RideRequest, driverID string, etaSec int) error {
	if c.broker == nil {
		return errors.New("broker is nil")
	}
	ev := dto.DriverAccepted{
		RideID:   req.RideID,
		DriverID: driverID,
		ETA:      etaSec,
	}
	routing := fmt.Sprintf("driver.response.%s", req.RideID)
	return c.broker.PublishJSON(c.appCtx, driverTopic, routing, ev)
}

func (c *MatchingConsumer) publishNoDriver(req dto.RideRequest, reason string) error {
	if c.broker == nil {
		return errors.New("broker is nil")
	}
	ev := dto.RideNoDriver{RideID: req.RideID, Reason: reason}
	return c.broker.PublishJSON(c.appCtx, rideExchange, "ride.status.NO_DRIVER", ev)
}

func (c *MatchingConsumer) initBroker() error {
	if c.cfg.RabbitMq == nil {
		return fmt.Errorf("missing RabbitMq config")
	}
	mbClient, err := bm.New(c.appCtx, *c.cfg.RabbitMq, c.log)
	if err != nil {
		return fmt.Errorf("broker connect: %w", err)
	}
	c.broker = mbClient
	return nil
}
