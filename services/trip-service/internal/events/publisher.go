package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	ev "github.com/sezarsaman/intercity-sim/pkg/events"
	"github.com/sezarsaman/intercity-sim/pkg/mq"
)

func PublishTripRequested(ctx context.Context, p mq.Publisher, payload ev.TripRequested) error {
	env := ev.Envelope[ev.TripRequested]{
		Event:         "trip.requested",
		EventID:       uuid.NewString(),
		OccurredAt:    time.Now().UTC(),
		TraceID:       "",
		SchemaVersion: 1,
		Payload:       payload,
	}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return p.Publish(ctx, "trip.requested", b, map[string]any{"content-type": "application/json"})
}
