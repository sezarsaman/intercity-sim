package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTrip_Success(t *testing.T) {
	// stub pricing-service
	pricing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, `{"distance_km":10,"base":5,"per_km":1,"surge":1,"final":15}`); err != nil {
			t.Fatal(err)
		}
	}))
	defer pricing.Close()

	// stub trip-service
	trip := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(w, `{"id":"t-1","passenger_id":"p-1","status":"requested","quoted_price":15}`); err != nil {
			t.Fatal(err)
		}
	}))
	defer trip.Close()

	router := NewRouter(pricing.URL, trip.URL)

	gw := httptest.NewServer(router)
	defer gw.Close()

	req := TripGatewayRequest{
		PassengerID: "p-1",
		Origin:      Location{Lat: 1, Lng: 2, City: "A"},
		Destination: Location{Lat: 3, Lng: 4, City: "B"},
		VehicleType: "sedan",
	}
	body, _ := json.Marshal(req)

	res, err := http.Post(gw.URL+"/api/v1/trips", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	out, _ := io.ReadAll(res.Body)
	if !bytes.Contains(out, []byte(`"id":"t-1"`)) {
		t.Fatalf("unexpected body: %s", string(out))
	}
}

func TestCreateTrip_PricingUpstreamError(t *testing.T) {
	// pricing returns 500
	pricing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer pricing.Close()

	// trip should not be called; but stub anyway
	trip := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should-not-hit", http.StatusTeapot)
	}))
	defer trip.Close()

	router := NewRouter(pricing.URL, trip.URL)
	gw := httptest.NewServer(router)
	defer gw.Close()

	req := TripGatewayRequest{PassengerID: "p-1", Origin: Location{}, Destination: Location{}}
	body, _ := json.Marshal(req)

	res, err := http.Post(gw.URL+"/api/v1/trips", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", res.StatusCode)
	}
}
