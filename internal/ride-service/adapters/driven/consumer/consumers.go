package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/adapters/driven/bm"
	"ride-hail/internal/ride-service/core/domain/dto"
	"ride-hail/internal/ride-service/core/ports"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rideStatusQueue = "ride_status"
)

type Notification struct {
	cfg    *config.Config
	mylog  mylogger.Logger
	mb     ports.IRidesBroker
	ctx    context.Context
	appCtx context.Context

	mu sync.Mutex
	wg sync.WaitGroup
}

func NewNotification(
	ctx context.Context,
	appCtx context.Context,
	cfg *config.Config,
	mylog mylogger.Logger,
) *Notification {
	return &Notification{
		ctx:    ctx,
		appCtx: appCtx,
		cfg:    cfg,
		mylog:  mylog,
	}
}

// Run initializes worker that starts working. It returns when the worker stops.
func (n *Notification) Run() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	mylog := n.mylog.Action("Run-consumer")

	// Initialize RabbitMQ connection
	if err := n.initializeRabbitMQ(); err != nil {
		mylog.Action("mb_connection_failed").Error("Failed to connect to message broker", err)
		return err
	}
	mylog.Action("mb_connected").Info("Successful message broker connection")

	rideStatusBus, err := n.mb.ConsumeMessageFromDrivers(n.appCtx, rideStatusQueue, "")
	if err != nil {
		return fmt.Errorf("failed to consume order messages: %v", err)
	}
	go n.work(rideStatusBus, n.processOrderMsg)

	return nil
}

func (n *Notification) Stop(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.mylog.Action("graceful_shutdown_started").Info("Shutting down")

	// wait for all workers to finish
	n.wg.Wait()

	if n.mb != nil {
		if err := n.mb.Close(); err != nil {
			n.mylog.Action("mb_close_failed").Error("Failed to close message broker", err)
			return fmt.Errorf("mb close: %w", err)
		}
		n.mylog.Action("mb_closed").Info("Message broker closed")
	}

	n.mylog.Action("graceful_shutdown_completed").Info("Successfully shutted down")
	return nil
}

func (n *Notification) work(
	notifCh <-chan amqp.Delivery,
	processor func(amqp.Delivery) (error, bool),
) {
	for {
		select {
		case <-n.ctx.Done():
			n.mylog.Info("Stopping consumer due to shutdown")
			return
		case msg, ok := <-notifCh:
			if !ok {
				return
			}
			n.wg.Add(1)
			go func(msg amqp.Delivery) {
				defer n.wg.Done()
				if err, dlq := processor(msg); err != nil {
					n.mylog.Error("Failed to process message", err)
					if nackErr := msg.Nack(false, dlq); nackErr != nil {
						n.mylog.Error("Failed to nack message", nackErr)
					}
				}
			}(msg)
		}
	}
}

func (n *Notification) processOrderMsg(msg amqp.Delivery) (error, bool) {
	var ride dto.RideStatusUpdate
	if err := json.Unmarshal(msg.Body, &ride); err != nil {
		return fmt.Errorf("decode order status: %v", err), false
	}

	n.mylog.Info("Consumed order status message",
		"client_id", ride.ClientId,
		"ride_number", ride.RideNumber,
		"status", ride.Status,
	)
	// n.handleOrderStatus(order)

	if err := msg.Ack(false); err != nil {
		return fmt.Errorf("acknowledge message: %v", err), true
	}
	return nil, true
}

// // should not to send dead letter queue or not, this n.at is mean bool argument that return
// func (n *Notification) handleOrderStatus(order dto.OrderStatusUpdate) {
// 	n.hub.SendToUser("client:"+order.ClientId, dto.WsMessage{
// 		Type: "order_status_update",
// 		Data: order,
// 	})
// }

func (n *Notification) initializeRabbitMQ() error {
	mb, err := bm.New(n.appCtx, *n.cfg.RabbitMq, n.mylog)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %v", err)
	}
	n.mb = mb
	return nil
}
