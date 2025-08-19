package main

import "testing"

func TestHaversine(t *testing.T) {
	// Tehran (35.6892, 51.3890) to Mashhad (36.2605, 59.6168) ~ 741 km (straight line approx)
	d := haversine(35.6892, 51.3890, 36.2605, 59.6168)
	if d < 700 || d > 800 {
		t.Fatalf("unexpected distance: %f", d)
	}
}
