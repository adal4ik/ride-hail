package driven

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ConsumeOptions — опции для чтения из брокера
type ConsumeOptions struct {
	Prefetch     int  // сколько сообщений держим без ack
	AutoAck      bool // включить авто-подтверждение (лучше false)
	QueueDurable bool // очередь сохраняется при рестарте брокера
}

// IDriverBroker — общий интерфейс для работы с брокером сообщений
type IDriverBroker interface {
	// PublishJSON публикует объект как JSON в указанный exchange/routing key.
	PublishJSON(ctx context.Context, exchange, routingKey string, msg any) error

	// Consume подписывается на очередь с указанным биндингом.
	// Возвращает канал Deliveries (amqp.Delivery), из которого читает consumer.
	Consume(ctx context.Context, queueName, bindingKey string, opts ConsumeOptions) (<-chan amqp.Delivery, error)

	// IsAlive проверяет состояние соединения.
	IsAlive() bool

	// Close аккуратно закрывает канал и соединение.
	Close() error
}
