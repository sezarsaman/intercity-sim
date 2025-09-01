package eventpub

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	ev "github.com/sezarsaman/intercity-sim/pkg/events"
	"github.com/sezarsaman/intercity-sim/pkg/mq"
)

func PublishTripPriced(ctx context.Context, p mq.Publisher, tripID string, finalPrice float64, surge float64) error {
	env := ev.Envelope[ev.TripPriced]{
		Event:         "trip.priced",
		EventID:       uuid.NewString(),
		OccurredAt:    time.Now().UTC(),
		TraceID:       "",
		SchemaVersion: 1,
		Payload:       ev.TripPriced{TripID: tripID, FinalPrice: finalPrice, Surge: surge},
	}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return p.Publish(ctx, "trip.priced", b, map[string]any{"content-type": "application/json"})
}
