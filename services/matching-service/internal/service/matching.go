// matching.go
package service

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Matcher wraps a Redis client and the GEO key we use to store driver locations.
type Matcher struct {
	rdb *redis.Client
	key string // e.g. "drivers:geo"
}

// Match is a single nearby driver result with coordinates and distance (km).
type Match struct {
	DriverID  string
	Latitude  float64
	Longitude float64
	DistKm    float64
}

// NewMatcher creates a matcher with an existing go-redis client.
func NewMatcher(rdb *redis.Client, key string) *Matcher {
	if key == "" {
		key = "drivers:geo"
	}
	return &Matcher{rdb: rdb, key: key}
}

// NewMatcherFromURL is a convenience for building a client from a Redis URL,
// e.g. redis://:password@redis:6379/0
func NewMatcherFromURL(redisURL, key string) (*Matcher, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	// quick ping to fail fast on bad URL/conn
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return NewMatcher(rdb, key), nil
}

// UpsertDriver inserts or updates a driver's last known coordinates.
func (m *Matcher) UpsertDriver(ctx context.Context, driverID string, lat, lng float64) error {
	if driverID == "" {
		return errors.New("driverID is required")
	}
	// GeoAdd updates position if member exists.
	_, err := m.rdb.GeoAdd(ctx, m.key, &redis.GeoLocation{
		Name:      driverID,
		Longitude: lng,
		Latitude:  lat,
	}).Result()
	return err
}

// RemoveDriver removes a driver from the GEO set (e.g., when offline).
func (m *Matcher) RemoveDriver(ctx context.Context, driverID string) error {
	if driverID == "" {
		return errors.New("driverID is required")
	}
	// GEO is implemented on top of a sorted set, so ZREM is correct here.
	_, err := m.rdb.ZRem(ctx, m.key, driverID).Result()
	return err
}

// FindNearby returns up to `limit` closest drivers within `radiusKm` around (lat,lng).
// If limit <= 0, no Count is applied and Redis returns all within the radius.
// Results are sorted ascending (nearest first).
func (m *Matcher) FindNearby(ctx context.Context, lat, lng, radiusKm float64, limit int) ([]Match, error) {
	q := &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     radiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
		},
		WithCoord: true,
		WithDist:  true,
	}
	if limit > 0 {
		q.Count = limit
	}

	locs, err := m.rdb.GeoSearchLocation(ctx, m.key, q).Result()
	if err != nil {
		return nil, err
	}

	out := make([]Match, 0, len(locs))
	for _, d := range locs {
		out = append(out, Match{
			DriverID:  d.Name,
			Latitude:  d.Latitude,
			Longitude: d.Longitude,
			DistKm:    d.Dist, // already in km because RadiusUnit: "km"
		})
	}
	return out, nil
}
