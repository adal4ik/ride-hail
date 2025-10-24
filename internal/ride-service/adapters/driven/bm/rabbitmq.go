package bm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ride-hail/internal/config"
	"ride-hail/internal/mylogger"
	"ride-hail/internal/ride-service/core/ports"
	"sync"
	"time"

	messagebroker "ride-hail/internal/ride-service/core/domain/message_broker_dto"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchange       = "ride_topic"
	reconnInterval = 10
)

type RabbitMQ struct {
	ctx          context.Context
	cfg          config.RabbitMqconfig
	mylog        mylogger.Logger
	conn         *amqp.Connection
	ch           *amqp.Channel
	reconnecting bool
	mu           *sync.Mutex
}

// create RabbitMQ adapter
func New(ctx context.Context, rabbitmqCfg config.RabbitMqconfig, mylog mylogger.Logger) (ports.IRidesBroker, error) {
	r := &RabbitMQ{
		ctx:          ctx,
		cfg:          rabbitmqCfg,
		mylog:        mylog,
		mu:           &sync.Mutex{},
		reconnecting: false,
	}
	if err := r.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %v", err)
	}
	return r, nil
}

func (r *RabbitMQ) PushMessageToDrivers(ctx context.Context, message messagebroker.Ride) error {
	mylog := r.mylog.Action("pushMessage")

	if r.conn.IsClosed() {
		mylog.Error("connection between rabbitmq is closed", fmt.Errorf("closed conn"))
		go r.reconnect(r.ctx)
		return errors.New("connection is closed")
	}

	routingKey := fmt.Sprintf("ride.request.%s", message.RideType)
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Priority:     uint8(message.Priority),
		Body:         body,
	})
}

func (r *RabbitMQ) ConsumeMessageFromDrivers(ctx context.Context, queue, driverName string) (<-chan amqp.Delivery, error) {
	return r.ch.ConsumeWithContext(ctx, queue, driverName, false, false, false, false, nil)
}

func (r *RabbitMQ) IsAlive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if both connection and channel are initialized and not closed
	if r.conn == nil || r.conn.IsClosed() {
		return false
	}
	if r.ch == nil || r.ch.IsClosed() {
		return false
	}

	return true
}

func (r *RabbitMQ) Close() error {
	if r.ch != nil && !r.ch.IsClosed() {
		if err := r.ch.Close(); err != nil {
			return fmt.Errorf("close rabbitmq channel: %v", err)
		}
	}

	if r.conn != nil && !r.conn.IsClosed() {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("close rabbitmq connection: %v", err)
		}
	}
	return nil
}

// connect to rabbitmq
func (r *RabbitMQ) connect() error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%v:%v@%v:%v/%v",
		r.cfg.User,
		r.cfg.Password,
		r.cfg.Host,
		r.cfg.Port,
		r.cfg.VHost,
	))
	if err != nil {
		return err
	}

	// try channel
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	if err := ch.Confirm(false); err != nil {
		return err
	}
	r.conn = conn
	r.ch = ch
	return nil
}

func (r *RabbitMQ) reconnect(ctx context.Context) {
	r.mu.Lock()
	if r.reconnecting {
		r.mu.Unlock()
		return
	}
	r.reconnecting = true
	r.mu.Unlock()

	t := time.NewTicker(time.Second * reconnInterval)
	mylog := r.mylog.Action("mb_reconnecting")

	for {
		select {
		case <-t.C:
			if err := r.connect(); err == nil {
				t.Stop()
				mylog.Action("mb_reconnection_completed").Info("Successfully reconnected!")
				r.reconnecting = false
				return
			}
			mylog.Info("rabbitmq failed to reconnect")

		case <-ctx.Done():
			t.Stop()
			return
		}
	}
}
