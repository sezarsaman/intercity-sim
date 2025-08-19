package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

type Location struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	City string  `json:"city"`
}

type TripGatewayRequest struct {
	PassengerID string   `json:"passenger_id"`
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

type TripCreateRequest struct {
	PassengerID string   `json:"passenger_id"`
	Origin      Location `json:"origin"`
	Destination Location `json:"destination"`
	QuotedPrice float64  `json:"quoted_price"`
}

func main() {
	port := getenv("PORT", "8080")
	pricingURL := getenv("PRICING_URL", "http://localhost:8082")
	tripURL := getenv("TRIP_URL", "http://localhost:8081")

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Post("/api/v1/trips", func(w http.ResponseWriter, r *http.Request) {
		var req TripGatewayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 1) Ask pricing-service
		priceReqBody, _ := json.Marshal(map[string]any{
			"origin":       req.Origin,
			"destination":  req.Destination,
			"vehicle_type": req.VehicleType,
		})
		priceResp, err := http.Post(pricingURL+"/price", "application/json", bytes.NewReader(priceReqBody))
		if err != nil {
			http.Error(w, "pricing-service error: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer priceResp.Body.Close()
		if priceResp.StatusCode >= 300 {
			body, _ := io.ReadAll(priceResp.Body)
			http.Error(w, "pricing-service bad status: "+string(body), http.StatusBadGateway)
			return
		}
		var price PriceResponse
		if err := json.NewDecoder(priceResp.Body).Decode(&price); err != nil {
			http.Error(w, "pricing decode error: "+err.Error(), http.StatusBadGateway)
			return
		}

		// 2) Create trip in trip-service with quoted price
		tripCreate := TripCreateRequest{
			PassengerID: req.PassengerID,
			Origin:      req.Origin,
			Destination: req.Destination,
			QuotedPrice: price.Final,
		}
		tripBody, _ := json.Marshal(tripCreate)
		tripResp, err := http.Post(tripURL+"/trips", "application/json", bytes.NewReader(tripBody))
		if err != nil {
			http.Error(w, "trip-service error: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer tripResp.Body.Close()
		if tripResp.StatusCode >= 300 {
			body, _ := io.ReadAll(tripResp.Body)
			http.Error(w, "trip-service bad status: "+string(body), http.StatusBadGateway)
			return
		}
		// 3) Stream back the created trip
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "api-gateway")
		w.WriteHeader(http.StatusCreated)
		io.Copy(w, tripResp.Body)
	})

	log.Printf("[api-gateway] listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
