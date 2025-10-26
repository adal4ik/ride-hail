package bm

import (
	"context"
	"encoding/json"
	"fmt"

	"ride-hail/internal/driver-location-service/core/domain/dto"
	"ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	bindRideRequest = "ride.request.*"
	bindRideStatus  = "ride.status.*"
)

type Consumer struct {
	ctx    context.Context
	log    mylogger.Logger
	broker driven.IDriverBroker

	RideRequests chan dto.RideDetails
	RideStatuses chan dto.RideStatusUpdate
}

func NewConsumer(ctx context.Context, broker driven.IDriverBroker, log mylogger.Logger, RideRequests chan dto.RideDetails, RideStatuses chan dto.RideStatusUpdate) *Consumer {
	return &Consumer{
		ctx:          ctx,
		broker:       broker,
		log:          log,
		RideRequests: RideRequests,
		RideStatuses: RideStatuses,
	}
}

func (c *Consumer) ListenAll() error {
	reqMsgs, err := c.broker.Consume(
		c.ctx,
		"ride_requests",
		bindRideRequest,
		driven.ConsumeOptions{Prefetch: 20, AutoAck: false, QueueDurable: true},
	)
	if err != nil {
		return fmt.Errorf("consume ride.request: %w", err)
	}

	statusMsgs, err := c.broker.Consume(
		c.ctx,
		"ride_status",
		bindRideStatus,
		driven.ConsumeOptions{Prefetch: 20, AutoAck: false, QueueDurable: true},
	)
	if err != nil {
		return fmt.Errorf("consume ride.status: %w", err)
	}

	go c.listenRequests(reqMsgs)
	go c.listenStatuses(statusMsgs)

	c.log.Info("Consumers started for ride.request.* and ride.status.*")
	return nil
}

func (c *Consumer) listenRequests(msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-c.ctx.Done():
			c.log.Info("stop ride.request consumer: context done")
			return
		case msg, ok := <-msgs:
			if !ok {
				c.log.Info("ride.request channel closed")
				return
			}

			var req dto.RideDetails
			if err := json.Unmarshal(msg.Body, &req); err != nil {
				c.log.Action("consume.request").Error("failed to unmarshal ride request", err)
				_ = msg.Nack(false, false)
				continue
			}

			c.RideRequests <- req
			_ = msg.Ack(false)
			c.log.Action("consume.request").Info("received ride request",
				"ride_id", req.Ride_id, "type", req.Ride_type)
		}
	}
}

func (c *Consumer) listenStatuses(msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-c.ctx.Done():
			c.log.Info("stop ride.status consumer: context done")
			return
		case msg, ok := <-msgs:
			if !ok {
				c.log.Info("ride.status channel closed")
				return
			}

			var st dto.RideStatusUpdate
			if err := json.Unmarshal(msg.Body, &st); err != nil {
				c.log.Action("consume.status").Error("failed to unmarshal ride status", err)
				_ = msg.Nack(false, false)
				continue
			}

			c.RideStatuses <- st
			_ = msg.Ack(false)
			c.log.Action("consume.status").Info("received ride status",
				"ride_id", st.RideID, "status", st.Status)
		}
	}
}
