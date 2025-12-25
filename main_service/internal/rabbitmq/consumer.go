package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type HandlerFunc func(ctx context.Context, body []byte) error

type Consumer struct {
	ch  *amqp.Channel
	log *slog.Logger
}

func NewConsumer(ch *amqp.Channel, log *slog.Logger) *Consumer {
	return &Consumer{
		ch:  ch,
		log: log,
	}
}

func (c *Consumer) Consume(
	ctx context.Context,
	queue string,
	handler HandlerFunc,
) error {
	const op = "rabbitmq.Consume"

	if err := c.ch.Qos(
		1,
		0,
		false,
	); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	msgs, err := c.ch.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case msg, ok := <-msgs:
				if !ok {
					c.log.Warn(
						"consumer channel closed",
						slog.String("op", op),
					)
					return
				}

				if err := handler(ctx, msg.Body); err != nil {
					if err := msg.Nack(false, true); err != nil {
						c.log.Error(
							"nack failed",
							slog.String("op", op),
							slog.Any("error", err),
						)
					}
					continue
				}

				if err := msg.Ack(false); err != nil {
					c.log.Error(
						"ack failed",
						slog.String("op", op),
						slog.Any("error", err),
					)
				}
			}
		}
	}()

	return nil
}
