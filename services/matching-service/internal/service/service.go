package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	ev "github.com/sezarsaman/intercity-sim/pkg/events"
	"github.com/sezarsaman/intercity-sim/pkg/mq"
)

type Service struct {
	R   *redis.Client
	Pub mq.Publisher
}

func New(r *redis.Client, pub mq.Publisher) *Service {
	return &Service{R: r, Pub: pub}
}

// HandleTripPriced is an example handler you likely already wired:
// it looks up a nearby driver and publishes trip.matched.
// Keep it if you already implemented a version elsewhere.
func (s *Service) HandleTripPriced(ctx context.Context, msg ev.TripPriced) error {
	// TODO: call into your matching core to pick a driver from Redis Geo.
	// Below is just a placeholder to keep compilation happy if you haven’t finished the core yet.
	// Remove this stub once your real matching logic is in place.
	match := ev.TripMatched{
		TripID:     msg.TripID,
		DriverID:   "stub-driver",
		DriverLat:  35.70,
		DriverLng:  51.40,
		ETASeconds: 300,
	}

	if err := PublishTripMatched(ctx, s.Pub, match); err != nil {
		return fmt.Errorf("publish trip.matched: %w", err)
	}
	return nil
}

// ---- publisher (kept here for cohesion; move if you already have eventpub) ----
func PublishTripMatched(ctx context.Context, p mq.Publisher, payload ev.TripMatched) error {
	env := map[string]any{
		"name":    "trip.matched",
		"payload": payload,
	}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return p.Publish(ctx, "trip.matched", b, map[string]any{"content-type": "application/json"})
}
