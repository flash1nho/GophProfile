package broker

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sony/gobreaker"

	"github.com/flash1nho/GophProfile/pkg/utils"
	"go.opentelemetry.io/otel"
)

type Rabbit struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
	cb       *gobreaker.CircuitBreaker
}

func New(conn *amqp.Connection, ch *amqp.Channel) (*Rabbit, error) {
	r := &Rabbit{
		conn:     conn,
		ch:       ch,
		exchange: "avatars.exchange",
		cb:       utils.NewCircuitBreaker("rabbitmq"),
	}

	if err := ch.ExchangeDeclare(
		r.exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"avatars.queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if err := ch.QueueBind(
		q.Name,
		"avatar.uploaded",
		r.exchange,
		false,
		nil,
	); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Rabbit) Publish(ctx context.Context, event any) error {
	_, span := otel.Tracer("rabbitmq").Start(ctx, "PublishMessage")
	defer span.End()

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = r.cb.Execute(func() (any, error) {
		return nil, r.ch.Publish(
			r.exchange,
			"avatar.uploaded",
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
				DeliveryMode: amqp.Persistent,
			},
		)
	})

	return err
}

func (r *Rabbit) Ping() error {
	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("rabbitmq connection closed")
	}
	return nil
}
