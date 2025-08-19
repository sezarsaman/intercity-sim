package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPrice_NoSurge_Sedan(t *testing.T) {
	// 11:00 → no surge
	timeNow = func() time.Time { return time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC) }
	defer func() { timeNow = time.Now }()

	srv := httptest.NewServer(NewRouter())
	defer srv.Close()

	req := PriceRequest{
		Origin:      Location{Lat: 35.6892, Lng: 51.3890, City: "TEH"},
		Destination: Location{Lat: 36.2605, Lng: 59.6168, City: "MHD"},
		VehicleType: "sedan",
	}
	body, _ := json.Marshal(req)

	res, err := http.Post(srv.URL+"/price", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}

	var pr PriceResponse
	if err := json.NewDecoder(res.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	if pr.Surge != 1.0 {
		t.Fatalf("expected surge 1.0, got %v", pr.Surge)
	}
}

func TestPrice_Surge_VIP(t *testing.T) {
	// 08:00 → surge window
	timeNow = func() time.Time { return time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC) }
	defer func() { timeNow = time.Now }()

	srv := httptest.NewServer(NewRouter())
	defer srv.Close()

	req := PriceRequest{
		Origin:      Location{Lat: 35.6892, Lng: 51.3890, City: "TEH"},
		Destination: Location{Lat: 36.2605, Lng: 59.6168, City: "MHD"},
		VehicleType: "vip",
	}
	body, _ := json.Marshal(req)

	res, err := http.Post(srv.URL+"/price", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}

	var pr PriceResponse
	if err := json.NewDecoder(res.Body).Decode(&pr); err != nil {
		t.Fatal(err)
	}
	if pr.Surge < 1.39 || pr.Surge > 1.41 {
		t.Fatalf("expected surge ≈1.4, got %v", pr.Surge)
	}
}

func TestPrice_BadJSON(t *testing.T) {
	srv := httptest.NewServer(NewRouter())
	defer srv.Close()

	res, err := http.Post(srv.URL+"/price", "application/json", bytes.NewReader([]byte(`{bad json`)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}
