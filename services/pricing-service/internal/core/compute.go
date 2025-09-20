package core

import (
	"math"
	"strings"
	"time"
)

type Point struct{ Lat, Lng float64 }

type Input struct {
	Origin      Point
	Destination Point
	VehicleType string
	Now         time.Time
}

type Output struct {
	DistanceKm float64
	Base       float64
	PerKm      float64
	Surge      float64
	Final      float64
}

func Compute(in Input) Output {
	dist := haversine(in.Origin.Lat, in.Origin.Lng, in.Destination.Lat, in.Destination.Lng)

	base := 5.0
	perKm := 1.0
	surge := 1.0
	h := in.Now.UTC().Hour()
	if h >= 17 && h <= 20 {
		surge = 1.5
	}
	surge = surgeFor(in.Now, in.VehicleType)
	final := math.Round((base+perKm*dist)*surge*100) / 100

	return Output{
		DistanceKm: dist,
		Base:       base,
		PerKm:      perKm,
		Surge:      surge,
		Final:      final,
	}
}

func surgeFor(now time.Time, vehicleType string) float64 {
	h := now.Hour()
	peak := (h >= 7 && h <= 9) || (h >= 17 && h <= 20)
	if !peak {
		return 1.0
	}
	if strings.EqualFold(vehicleType, "vip") {
		return 1.4
	}
	return 1.2
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * (3.141592653589793 / 180.0)
	dLon := (lon2 - lon1) * (3.141592653589793 / 180.0)
	a := (sin2(dLat / 2)) + cosd(lat1)*cosd(lat2)*sin2(dLon/2)
	c := 2 * atan2sqrt(a)
	return R * c
}

func sin2(x float64) float64      { s := math.Sin(x); return s * s }
func cosd(d float64) float64      { return math.Cos(d * (3.141592653589793 / 180.0)) }
func atan2sqrt(a float64) float64 { return math.Atan2(math.Sqrt(a), math.Sqrt(1-a)) }
