package service

import (
	"context"
	"time"

	ev "github.com/sezarsaman/intercity-sim/pkg/events"
	"github.com/sezarsaman/intercity-sim/pkg/mq"
	"github.com/sezarsaman/intercity-sim/services/pricing-service/internal/core"
	"github.com/sezarsaman/intercity-sim/services/pricing-service/internal/eventpub"
)

type PricingService struct{ Pub mq.Publisher }

func New(pub mq.Publisher) *PricingService { return &PricingService{Pub: pub} }

func (s *PricingService) HandleTripRequested(ctx context.Context, e ev.TripRequested) error {
	out := core.Compute(core.Input{
		Origin:      core.Point{Lat: e.Origin.Lat, Lng: e.Origin.Lng},
		Destination: core.Point{Lat: e.Destination.Lat, Lng: e.Destination.Lng},
		VehicleType: e.VehicleType,
		Now:         time.Now(),
	})
	return eventpub.PublishTripPriced(ctx, s.Pub, e.TripID, out.Final, out.Surge)
}
