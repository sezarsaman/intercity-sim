package mq

import (
	"context"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbit struct{ conn *amqp.Connection }

func Dial(url string) (*Rabbit, error) {
	c, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return &Rabbit{conn: c}, nil
}
func (r *Rabbit) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

type rabbitPublisher struct {
	ch       *amqp.Channel
	exchange string
}

func (r *Rabbit) Publisher(exchange string) (Publisher, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return nil, err
	}
	return &rabbitPublisher{ch: ch, exchange: exchange}, nil
}

func (p *rabbitPublisher) Publish(ctx context.Context, routingKey string, body []byte, headers map[string]any) error {
	if p.ch == nil {
		return errors.New("publisher channel is nil")
	}
	h := amqp.Table{}
	for k, v := range headers {
		h[k] = v
	}
	return p.ch.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		Headers:      h,
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	})
}

type rabbitSubscriber struct {
	ch              *amqp.Channel
	exchange, queue string
	prefetch        int
}

func (r *Rabbit) Subscriber(exchange, queue string, prefetch int) (Subscriber, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if prefetch > 0 {
		if err := ch.Qos(prefetch, 0, false); err != nil {
			_ = ch.Close()
			return nil, err
		}
	}
	return &rabbitSubscriber{ch: ch, exchange: exchange, queue: queue, prefetch: prefetch}, nil
}

func (s *rabbitSubscriber) Consume(ctx context.Context, bindingKey string, handler func(ctx context.Context, body []byte, headers map[string]any) error) error {
	if err := s.ch.QueueBind(s.queue, bindingKey, s.exchange, false, nil); err != nil {
		return err
	}
	msgs, err := s.ch.Consume(s.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return nil
			}
			h := map[string]any{}
			for k, v := range d.Headers {
				h[k] = v
			}
			if err := handler(ctx, d.Body, h); err != nil {
				_ = d.Nack(false, true)
			} else {
				_ = d.Ack(false)
			}
		}
	}
}
