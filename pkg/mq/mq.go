package mq

import "context"

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte, headers map[string]any) error
}

type Subscriber interface {
	Consume(ctx context.Context, bindingKey string, handler func(ctx context.Context, body []byte, headers map[string]any) error) error
}
