package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Post("/price", func(w http.ResponseWriter, r *http.Request) {
		var req PriceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		dist := haversine(req.Origin.Lat, req.Origin.Lng, req.Destination.Lat, req.Destination.Lng)

		// simple pricing model
		base := 5.0
		perKm := 0.8
		// crude surge by hour of day
		h := time.Now().Hour()
		surge := 1.0
		if (h >= 7 && h <= 9) || (h >= 17 && h <= 20) {
			surge = 1.3
		}
		if req.VehicleType == "vip" {
			perKm = 1.2
			base = 8.0
			surge += 0.1
		}

		final := (base + perKm*dist) * surge
		final = math.Round(final*100) / 100

		resp := PriceResponse{
			DistanceKm: round2(dist),
			Base:       base,
			PerKm:      perKm,
			Surge:      surge,
			Final:      final,
		}
		writeJSON(w, http.StatusOK, resp)
	})

	log.Printf("[pricing-service] listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Service", "pricing-service")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	// Earth radius km
	const R = 6371.0
	// deg2rad
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// optional: parse env float
func getenvFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}
