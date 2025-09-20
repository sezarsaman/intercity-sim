package events

import (
	"context"
)

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher { return &NoopPublisher{} }

func (n *NoopPublisher) Publish(ctx context.Context, routingKey string, body []byte, headers map[string]any) error {
	return nil
}
