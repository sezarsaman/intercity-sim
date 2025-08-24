package main

import (
	"context"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"io"
	"net/http"
	"net/http/httptest"
)

func setupPostgres(t *testing.T) (func(), string) {
	ctx := context.Background()

	req := tc.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "tripdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	pgC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	host, err := pgC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := pgC.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}

	dsn := "postgres://postgres:postgres@" + host + ":" + port.Port() + "/tripdb?sslmode=disable"

	// cleanup
	tearDown := func() {
		pgC.Terminate(ctx)
	}
	return tearDown, dsn
}

func TestInsertAndGetTrip(t *testing.T) {
	tearDown, dsn := setupPostgres(t)
	defer tearDown()

	ctx := context.Background()

	t.Setenv("DATABASE_URL", dsn)
	pool, err := connectDBFromEnv(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	req := Trip{
		PassengerID: "p1",
		Origin:      Location{Lat: 35.7, Lng: 51.4, City: "Tehran"},
		Destination: Location{Lat: 32.6, Lng: 51.7, City: "Isfahan"},
		QuotedPrice: 250000,
	}

	trip, err := insertTrip(ctx, pool, req)
	if err != nil {
		t.Fatalf("insertTrip failed: %v", err)
	}

	got, ok, err := getTripByID(ctx, pool, trip.ID)
	if err != nil {
		t.Fatalf("getTripByID failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected trip to exist, not found")
	}

	if got.PassengerID != req.PassengerID || got.Origin.City != req.Origin.City {
		t.Errorf("mismatch in trip data: got %+v, want %+v", got, req)
	}
}

func TestHealthDBEndpoint(t *testing.T) {
	tearDown, dsn := setupPostgres(t)
	defer tearDown()

	ctx := context.Background()
	t.Setenv("DATABASE_URL", dsn)
	pool, err := connectDBFromEnv(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(NewRouter(pool))
	defer server.Close()

	resp, err := http.Get(server.URL + "/health/db")
	if err != nil {
		t.Fatalf("failed to call /health/db: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK, got %d, body=%s", resp.StatusCode, body)
	}
}
