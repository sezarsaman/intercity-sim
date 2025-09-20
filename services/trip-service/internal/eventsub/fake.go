package eventsub

import "context"

type NoopSubscriber struct{}

func NewNoopSubscriber() *NoopSubscriber { return &NoopSubscriber{} }

func (n *NoopSubscriber) Consume(ctx context.Context, bindingKey string, handler func(ctx context.Context, body []byte, headers map[string]any) error) error {
	<-ctx.Done()
	return ctx.Err()
}
