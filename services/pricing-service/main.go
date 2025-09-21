package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	chi "github.com/go-chi/chi/v5"

	"os/signal"
	"syscall"

	"github.com/sezarsaman/intercity-sim/pkg/mq"
	eventsub "github.com/sezarsaman/intercity-sim/services/pricing-service/internal/eventsub"
	psvc "github.com/sezarsaman/intercity-sim/services/pricing-service/internal/service"

	core "github.com/sezarsaman/intercity-sim/services/pricing-service/internal/core"
)

type Location struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	City string  `json:"city"`
}

type PriceRequest struct {
	Origin      Location `json:"origin"`
	Destination Location `json:"destination"`
	VehicleType string   `json:"vehicle_type,omitempty"`
}

type PriceResponse struct {
	DistanceKm float64 `json:"distance_km"`
	Base       float64 `json:"base"`
	PerKm      float64 `json:"per_km"`
	Surge      float64 `json:"surge"`
	Final      float64 `json:"final"`
}

// injectable clock for tests
var timeNow = time.Now

func main() {
	port := getenv("PORT", "8082")
	r := NewRouter()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rabbitURL := getenv("RABBIT_URL", "amqp://guest:guest@rabbitmq:5672/")

	if err := mq.BootstrapTopology(rabbitURL); err != nil {
		log.Fatalf("pricing-service: topology bootstrap failed: %v", err)
	}

	rb, err := mq.Dial(rabbitURL)
	if err != nil {
		log.Fatal(err)
	}
	defer rb.Close()

	pub, err := rb.Publisher("rides.events")
	if err != nil {
		log.Fatal(err)
	}

	sub, err := rb.Subscriber(mq.ExchangeEvents, mq.QueueTripRequested, 10)
	if err != nil {
		log.Fatal(err)
	}

	svc := psvc.New(pub)
	go func() {
		if err := eventsub.ConsumeTripRequested(ctx, sub, svc.HandleTripRequested); err != nil {
			log.Printf("pricing-service: consumer stopped: %v", err)
		}
	}()

	log.Printf("[pricing-service] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))

}

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("pricing-service: write health response error: %v", err)
		}
	})

	r.Post("/price", func(w http.ResponseWriter, r *http.Request) {

		var req struct {
			Origin struct {
				Lat  float64 `json:"lat"`
				Lng  float64 `json:"lng"`
				City string  `json:"city"`
			} `json:"origin"`
			Destination struct {
				Lat  float64 `json:"lat"`
				Lng  float64 `json:"lng"`
				City string  `json:"city"`
			} `json:"destination"`
			VehicleType string `json:"vehicle_type"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}

		out := core.Compute(core.Input{
			Origin:      core.Point{Lat: req.Origin.Lat, Lng: req.Origin.Lng},
			Destination: core.Point{Lat: req.Destination.Lat, Lng: req.Destination.Lng},
			VehicleType: req.VehicleType,
			Now:         timeNow(),
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"distance_km": out.DistanceKm,
			"base":        out.Base,
			"per_km":      out.PerKm,
			"surge":       out.Surge,
			"final":       out.Final,
		}); err != nil {
			log.Printf("pricing-service: encode error: %v", err)
		}
	})

	return r
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1 = lat1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
