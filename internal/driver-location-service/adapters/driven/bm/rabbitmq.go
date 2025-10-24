package bm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"ride-hail/internal/config"
	ports "ride-hail/internal/driver-location-service/core/ports/driven"
	"ride-hail/internal/mylogger"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rideExchangeName = "ride_topic" // topic
	reconnInterval   = 5            // seconds
)

type RabbitMQ struct {
	ctx          context.Context
	cfg          config.RabbitMqconfig
	log          mylogger.Logger
	conn         *amqp.Connection
	ch           *amqp.Channel
	reconnecting bool
	mu           *sync.Mutex
}

var _ ports.IDriverBroker = (*RabbitMQ)(nil)

func New(ctx context.Context, rabbitmqCfg config.RabbitMqconfig, log mylogger.Logger) (ports.IDriverBroker, error) {
	r := &RabbitMQ{
		ctx:          ctx,
		cfg:          rabbitmqCfg,
		log:          log,
		mu:           &sync.Mutex{},
		reconnecting: false,
	}
	if err := r.connect(); err != nil {
		return nil, fmt.Errorf("rabbit connect: %w", err)
	}
	return r, nil
}

func (r *RabbitMQ) PublishJSON(ctx context.Context, exchange, routingKey string, msg any) error {
	if !r.IsAlive() {
		r.log.Action("publish").Error("amqp not alive", errors.New("amqp closed"))
		go r.reconnect(r.ctx)
		return errors.New("amqp closed")
	}
	if err := r.ensureExchange(exchange); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	pubctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return r.ch.PublishWithContext(pubctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (r *RabbitMQ) Consume(ctx context.Context, queueName, bindingKey string, opts ports.ConsumeOptions) (<-chan amqp.Delivery, error) {
	if !r.IsAlive() {
		return nil, errors.New("amqp closed")
	}
	// гарантируем exchange ride_topic (мы читаем из него по биндингу)
	if err := r.ensureExchange(rideExchangeName); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}
	// очередь
	_, err := r.ch.QueueDeclare(
		queueName,
		opts.QueueDurable,
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("queue declare: %w", err)
	}
	// биндинг очереди к ride_topic по ключу
	if err := r.ch.QueueBind(
		queueName,
		bindingKey,
		rideExchangeName,
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("queue bind: %w", err)
	}
	// prefetch
	if opts.Prefetch > 0 {
		if err := r.ch.Qos(opts.Prefetch, 0, false); err != nil {
			return nil, fmt.Errorf("qos: %w", err)
		}
	}
	// consume из очереди
	deliveries, err := r.ch.Consume(
		queueName,
		"",           // consumer tag
		opts.AutoAck, // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	out := make(chan amqp.Delivery)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-deliveries:
				if !ok {
					return
				}
				out <- m
			}
		}
	}()
	return out, nil
}

func (r *RabbitMQ) IsAlive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
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
			return fmt.Errorf("close channel: %w", err)
		}
	}
	if r.conn != nil && !r.conn.IsClosed() {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("close connection: %w", err)
		}
	}
	return nil
}

func (r *RabbitMQ) ensureExchange(name string) error {
	return r.ch.ExchangeDeclare(name, "topic", true, false, false, false, nil)
}

func (r *RabbitMQ) connect() error {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		r.cfg.User, r.cfg.Password, r.cfg.Host, r.cfg.Port, r.cfg.VHost,
	)
	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	// включаем publisher confirms (ожидание можно добавить позже)
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	// заранее объявим ride_topic
	if err := ch.ExchangeDeclare(rideExchangeName, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
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

	t := time.NewTicker(time.Duration(reconnInterval) * time.Second)
	l := r.log.Action("mb_reconnecting")

	for {
		select {
		case <-t.C:
			if err := r.connect(); err == nil {
				t.Stop()
				l.Action("mb_reconnection_completed").Info("reconnected")
				r.reconnecting = false
				return
			}
			l.Info("reconnect failed")
		case <-ctx.Done():
			t.Stop()
			return
		}
	}
}
