package bm

import (
	"context"
	ports "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"
)

type Publisher struct {
	ctx    context.Context
	log    mylogger.Logger
	broker ports.IDriverBroker
}

func NewPublisher(ctx context.Context, broker ports.IDriverBroker, log mylogger.Logger) *Publisher {
	return &Publisher{
		ctx:    ctx,
		broker: broker,
		log:    log,
	}
}

func (p *Publisher) PublishMessage(subject string, routingKey string, msg any) error {
	err := p.broker.PublishJSON(p.ctx, rideExchangeName, routingKey, msg)
	if err != nil {
		p.log.Action("publish").Error("failed to publish message", err)
		return err
	}
	p.log.Action("publish").Info("message published successfully", nil)
	return nil
}
