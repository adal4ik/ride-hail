package bm

import (
	"context"
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

func NewConsumer(ctx context.Context, broker driven.IDriverBroker, log mylogger.Logger) *Consumer {
	return &Consumer{
		ctx:    ctx,
		broker: broker,
		log:    log,
	}
}

func (c *Consumer) ListenAll() (<-chan amqp.Delivery, <-chan amqp.Delivery, error) {
	reqMsgs, err := c.broker.Consume(
		c.ctx,
		"ride_requests",
		bindRideRequest,
		driven.ConsumeOptions{Prefetch: 20, AutoAck: false, QueueDurable: true},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("consume ride.request: %w", err)
	}

	statusMsgs, err := c.broker.Consume(
		c.ctx,
		"ride_status",
		bindRideStatus,
		driven.ConsumeOptions{Prefetch: 20, AutoAck: false, QueueDurable: true},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("consume ride.status: %w", err)
	}
	c.log.Info("Consumers started for ride.request.* and ride.status.*")
	return reqMsgs, statusMsgs, nil
}
