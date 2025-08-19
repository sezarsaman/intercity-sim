package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Location struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	City string  `json:"city"`
}

type Trip struct {
	ID              string   `json:"id"`
	PassengerID     string   `json:"passenger_id"`
	Origin          Location `json:"origin"`
	Destination     Location `json:"destination"`
	Status          string   `json:"status"`
	QuotedPrice     float64  `json:"quoted_price"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TripCreateRequest struct {
	PassengerID string   `json:"passenger_id"`
	Origin      Location `json:"origin"`
	Destination Location `json:"destination"`
	QuotedPrice float64  `json:"quoted_price"`
}

func main() {
	port := getenv("PORT", "8081")
	dbURL := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/tripdb?sslmode=disable")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	if err := migrate(ctx, pool); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Post("/trips", func(w http.ResponseWriter, r *http.Request) {
		var req TripCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id := uuid.NewString()
		now := time.Now().UTC()
		status := "requested"

		_, err := pool.Exec(r.Context(), `
			INSERT INTO trips (
				id, passenger_id,
				origin_lat, origin_lng, origin_city,
				destination_lat, destination_lng, destination_city,
				status, quoted_price, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`,
			id, req.PassengerID,
			req.Origin.Lat, req.Origin.Lng, req.Origin.City,
			req.Destination.Lat, req.Destination.Lng, req.Destination.City,
			status, req.QuotedPrice, now, now,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		trip, err := getTrip(r.Context(), pool, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, trip)
	})

	r.Get("/trips/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		trip, err := getTrip(r.Context(), pool, id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, trip)
	})

	log.Printf("[trip-service] listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

var ErrNotFound = errors.New("not found")

func getTrip(ctx context.Context, pool *pgxpool.Pool, id string) (Trip, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, passenger_id,
		       origin_lat, origin_lng, origin_city,
		       destination_lat, destination_lng, destination_city,
		       status, quoted_price, created_at, updated_at
		FROM trips WHERE id=$1
	`, id)
	var t Trip
	err := row.Scan(
		&t.ID, &t.PassengerID,
		&t.Origin.Lat, &t.Origin.Lng, &t.Origin.City,
		&t.Destination.Lat, &t.Destination.Lng, &t.Destination.City,
		&t.Status, &t.QuotedPrice, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return Trip{}, ErrNotFound
		}
		return Trip{}, err
	}
	return t, nil
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS trips (
  id TEXT PRIMARY KEY,
  passenger_id TEXT NOT NULL,
  origin_lat DOUBLE PRECISION NOT NULL,
  origin_lng DOUBLE PRECISION NOT NULL,
  origin_city TEXT NOT NULL,
  destination_lat DOUBLE PRECISION NOT NULL,
  destination_lng DOUBLE PRECISION NOT NULL,
  destination_city TEXT NOT NULL,
  status TEXT NOT NULL,
  quoted_price DOUBLE PRECISION NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`)
	return err
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Service", "trip-service")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}
