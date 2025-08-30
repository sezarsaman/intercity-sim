package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	eventpub "github.com/sezarsaman/intercity-sim/services/trip-service/internal/events"
	"github.com/sezarsaman/intercity-sim/services/trip-service/internal/eventsub"
	svc "github.com/sezarsaman/intercity-sim/services/trip-service/internal/service"
)

// ========== Domain Models ==========

type Location struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	City string  `json:"city"`
}

type Trip struct {
	ID          string    `json:"id"`
	PassengerID string    `json:"passenger_id"`
	Origin      Location  `json:"origin"`
	Destination Location  `json:"destination"`
	QuotedPrice float64   `json:"quoted_price"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type TripCreateRequest struct {
	PassengerID string   `json:"passenger_id"`
	Origin      Location `json:"origin"`
	Destination Location `json:"destination"`
	QuotedPrice float64  `json:"quoted_price"`
}

// ========== DB Wiring ==========

func connectDBFromEnv(ctx context.Context) (*pgxpool.Pool, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		cfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
		}
		cfg.MaxConns = 4
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("create pool (DATABASE_URL): %w", err)
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("db ping (DATABASE_URL) failed: %w", err)
		}
		return pool, nil
	}

	dbHost := getenv("DB_HOST", "tripdb")
	dbPort := getenv("DB_PORT", "5432")
	dbUser := getenv("DB_USER", "postgres")
	dbPassword := getenv("DB_PASSWORD", "postgres")
	dbName := getenv("DB_NAME", "tripdb")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 4

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db ping failed: %w", err)
	}
	return pool, nil
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS trips (
		id            TEXT PRIMARY KEY,
		passenger_id  TEXT        NOT NULL,
		origin_lat    DOUBLE PRECISION NOT NULL,
		origin_lng    DOUBLE PRECISION NOT NULL,
		origin_city   TEXT,
		dest_lat      DOUBLE PRECISION NOT NULL,
		dest_lng      DOUBLE PRECISION NOT NULL,
		dest_city     TEXT,
		quoted_price  NUMERIC(10,2)    NOT NULL,
		status        TEXT        NOT NULL DEFAULT 'requested',
		created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_trips_passenger ON trips (passenger_id);
		CREATE INDEX IF NOT EXISTS idx_trips_created_at ON trips (created_at DESC);
		`
	if _, err := pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("migrate: exec DDL: %w", err)
	}
	return nil
}

// ========== Data Access (CRUD) ==========

func insertTrip(ctx context.Context, pool *pgxpool.Pool, req Trip) (*Trip, error) {
	id := uuid.New().String()

	const q = `
		INSERT INTO trips (
		id, passenger_id,
		origin_lat, origin_lng, origin_city,
		dest_lat, dest_lng, dest_city,
		quoted_price
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING status, created_at
		`

	var t Trip
	t.ID = id
	t.PassengerID = req.PassengerID
	t.Origin = req.Origin
	t.Destination = req.Destination
	t.QuotedPrice = req.QuotedPrice

	if err := pool.QueryRow(
		ctx, q,
		t.ID, t.PassengerID,
		t.Origin.Lat, t.Origin.Lng, t.Origin.City,
		t.Destination.Lat, t.Destination.Lng, t.Destination.City,
		t.QuotedPrice,
	).Scan(&t.Status, &t.CreatedAt); err != nil {
		return nil, fmt.Errorf("insertTrip failed: %w", err)
	}

	return &t, nil
}

func getTripByID(ctx context.Context, pool *pgxpool.Pool, id string) (Trip, bool, error) {
	const q = `
		SELECT id, passenger_id,
			origin_lat, origin_lng, origin_city,
			dest_lat, dest_lng, dest_city,
			quoted_price, status, created_at
		FROM trips
		WHERE id = $1
		`

	var t Trip
	var origin Location
	var dest Location

	err := pool.QueryRow(ctx, q, id).Scan(
		&t.ID,
		&t.PassengerID,
		&origin.Lat, &origin.Lng, &origin.City,
		&dest.Lat, &dest.Lng, &dest.City,
		&t.QuotedPrice,
		&t.Status,
		&t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Trip{}, false, nil
		}
		return Trip{}, false, fmt.Errorf("getTripByID failed: %w", err)
	}

	t.Origin = origin
	t.Destination = dest

	return t, true, nil
}

// ========== HTTP (Router + Handlers) ==========

func NewRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("trip-service: write health response error: %v", err)
		}
	})

	r.Get("/health/db", func(w http.ResponseWriter, r *http.Request) {
		if _, err := pool.Exec(r.Context(), "SELECT 1"); err != nil {
			http.Error(w, "db not ok: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("trip-service: write health response error: %v", err)
		}
	})

	r.Post("/trips", createTripHandler(pool))
	r.Get("/trips/{id}", getTripHandler(pool))

	return r
}

func createTripHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Trip
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.PassengerID == "" {
			http.Error(w, "passenger_id is required", http.StatusBadRequest)
			return
		}

		t, err := insertTrip(r.Context(), pool, req)
		if err != nil {
			http.Error(w, "db insert error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		pub := eventpub.NewNoopPublisher()
		go func(id, pid string) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := eventpub.PublishTripRequested(ctx, pub, id, pid); err != nil {
				log.Printf("publish trip.requested error: %v", err)
			}
		}(t.ID, t.PassengerID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(t); err != nil {
			log.Printf("encode response error: %v", err)
		}
	}
}

func getTripHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		t, ok, err := getTripByID(r.Context(), pool, id)
		if err != nil {
			http.Error(w, "db select error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(t)
	}
}

// ========== Bootstrap ==========

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectDBFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}

	h := svc.NewPricedHandler(pool)
	sub := eventsub.NewNoopSubscriber()
	go func() {
		if err := eventsub.ConsumeTripPriced(ctx, sub, h.Handle); err != nil {
			log.Printf("trip-service: consumer stopped: %v", err)
		}
	}()

	port := getenv("PORT", "8081")
	log.Printf("[trip-service] listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, NewRouter(pool)))
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
