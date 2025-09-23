package service

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ev "github.com/sezarsaman/intercity-sim/pkg/events"
)

type MatchedHandler struct {
	pool *pgxpool.Pool
}

func NewMatchedHandler(pool *pgxpool.Pool) *MatchedHandler {
	return &MatchedHandler{pool: pool}
}

func (h *MatchedHandler) HandleTripMatched(ctx context.Context, msg ev.TripMatched, eventID string) error {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		// Rollback only if not already committed
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			// log it, but don't override the main error
			log.Printf("trip-service: rollback error: %v", err)
		}
	}()

	// idempotency inbox
	if _, err := tx.Exec(ctx,
		`INSERT INTO event_inbox(event_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		eventID,
	); err != nil {
		return fmt.Errorf("inbox upsert: %w", err)
	}

	// update trip as matched (tweak columns to your schema)
	if _, err := tx.Exec(ctx,
		`UPDATE trips
		   SET status='matched', driver_id=$2, driver_lat=$3, driver_lng=$4, eta_seconds=$5
		 WHERE id=$1`,
		msg.TripID, msg.DriverID, msg.DriverLat, msg.DriverLng, msg.ETASeconds,
	); err != nil {
		return fmt.Errorf("update trip matched: %w", err)
	}

	return tx.Commit(ctx)
}
