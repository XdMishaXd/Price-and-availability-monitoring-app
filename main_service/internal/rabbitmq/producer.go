package rabbitmq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer struct {
	ch *amqp.Channel
}

func NewProducer(ch *amqp.Channel) *Producer {
	return &Producer{ch: ch}
}

func (p *Producer) PublishJSON(
	ctx context.Context,
	queue string,
	msg any,
) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(
		ctx,
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
