package broker

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbit struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
}

func New(ch *amqp.Channel) (*Rabbit, error) {
	r := &Rabbit{
		ch:       ch,
		exchange: "avatars.exchange",
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

func (r *Rabbit) Publish(event any) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return r.ch.Publish(
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
}

func (r *Rabbit) Ping() error {
	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("rabbitmq connection closed")
	}
	return nil
}
