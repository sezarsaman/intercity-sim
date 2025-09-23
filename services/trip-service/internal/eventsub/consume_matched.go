package eventsub

import (
	"context"
	"encoding/json"

	ev "github.com/sezarsaman/intercity-sim/pkg/events"
	"github.com/sezarsaman/intercity-sim/pkg/mq"
)

func ConsumeTripMatched(
	ctx context.Context,
	sub mq.Subscriber,
	on func(context.Context, ev.TripMatched, string) error,
) error {
	return sub.Consume(ctx, "trip.matched", func(c context.Context, body []byte, headers map[string]any) error {
		var env ev.Envelope[ev.TripMatched]
		if err := json.Unmarshal(body, &env); err != nil {
			return err
		}
		return on(c, env.Payload, env.EventID) // <-- important
	})
}
