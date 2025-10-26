package bm

import (
	"context"
	"encoding/json"
	"fmt"
	"ride-hail/internal/driver-location-service/core/domain/dto"
	ports "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"
	"time"
)

type IConsumer interface {
	SubscribeForMessages() error
}

type Consumer struct {
	ctx      context.Context
	log      mylogger.Logger
	broker   ports.IDriverBroker
	messages chan dto.RideDetails
}

func NewConsumer(ctx context.Context, broker ports.IDriverBroker, log mylogger.Logger, messages chan dto.RideDetails) *Consumer {
	return &Consumer{
		ctx:      ctx,
		broker:   broker,
		log:      log,
		messages: messages,
	}
}

func (c *Consumer) SubscribeForMessages() error {
	time.Sleep(2 * time.Second)
	msgCh, err := c.broker.Consume(c.ctx, "ride_requests", "ride.request.{ride_type}", ports.ConsumeOptions{
		Prefetch:     1,
		AutoAck:      false,
		QueueDurable: true,
	})
	if err != nil {
		c.log.Action("consume").Error("messages channel closed", nil)
		return err
	}
	go func() {
		for msg := range msgCh {
			var unmarshedMsg dto.RideDetails
			fmt.Println("Received message in Consumer Messsage Content Type: ", msg.ContentType)
			if err := json.Unmarshal(msg.Body, &unmarshedMsg); err != nil {
				c.log.Action("consume").Error("failed to unmarshal message", err)
				continue
			}
			fmt.Println("Unmarshalled message: ", unmarshedMsg)
			c.messages <- unmarshedMsg
			c.log.Action("consume").Info("message received", nil)
			// Process the message
			c.log.Action("consume").Info("message body: %s", string(msg.Body))
			// Acknowledge the message
			if err := msg.Ack(false); err != nil {
				c.log.Action("consume").Error("failed to acknowledge message", err)
			} else {
				c.log.Action("consume").Info("message acknowledged", nil)
			}
		}
	}()
	return nil
}
