package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"

	bm "ride-hail/internal/driver-location-service/adapters/driven/bm"
	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/ports/driven"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	driverMatchingQ = "driver_matching"
	rideRequestBind = "ride.request.*"
	defaultPrefetch = 10
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

	log := c.log.Action("driver-matching-run")

	// Подключаемся к RabbitMQ
	if err := c.initBroker(); err != nil {
		log.Error("rabbitmq connect failed", err)
		return err
	}
	log.Info("rabbitmq connected")

	// Подписываемся на очередь ride.request.*
	msgs, err := c.broker.Consume(
		c.appCtx,
		driverMatchingQ,
		rideRequestBind,
		driven.ConsumeOptions{Prefetch: defaultPrefetch, AutoAck: true, QueueDurable: true},
	)
	if err != nil {
		return fmt.Errorf("consume driver_matching: %w", err)
	}

	// Слушаем и просто печатаем
	go c.listen(msgs)

	log.Info("driver-matching consumer started")
	return nil
}

func (c *MatchingConsumer) listen(msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-c.ctx.Done():
			c.log.Info("stop consumer: context done")
			return
		case msg, ok := <-msgs:
			if !ok {
				c.log.Info("stop consumer: channel closed")
				return
			}

			var req dto.RideRequest
			if err := json.Unmarshal(msg.Body, &req); err != nil {
				c.log.Error("failed to decode ride.request", err)
				continue
			}

			fmt.Printf("[📨] New ride request: ID=%s, Type=%s, Passenger=%s, Pickup=(%.4f, %.4f)\n",
				req.RideID, req.RideType, req.PassengerID, req.Pickup.Lat, req.Pickup.Lng)
		}
	}
}

func (c *MatchingConsumer) Stop(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Action("shutdown").Info("graceful shutdown started")
	c.wg.Wait()

	if c.broker != nil {
		if err := c.broker.Close(); err != nil {
			c.log.Error("rabbit close failed", err)
			return err
		}
	}
	c.log.Info("graceful shutdown done")
	return nil
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
