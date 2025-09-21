package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	ev "github.com/sezarsaman/intercity-sim/pkg/events"
)

type PricedHandler struct {
	pool *pgxpool.Pool
}

func NewPricedHandler(pool *pgxpool.Pool) *PricedHandler {
	return &PricedHandler{pool: pool}
}

func (h *PricedHandler) Handle(ctx context.Context, payload ev.TripPriced, eventID string) error {
	eventID, ok := ev.EventIDFrom(ctx)
	if !ok || eventID == "" {
		// If the envelope was malformed we can still be safe by refusing to apply state.
		return errors.New("missing event_id in context")
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// 1) Inbox dedupe (idempotency)
	tag, err := tx.Exec(ctx,
		`INSERT INTO event_inbox(event_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		eventID,
	)
	if err != nil {
		return fmt.Errorf("inbox upsert: %w", err)
	}
	// If 0 rows inserted → we’ve seen this event before, do nothing = idempotent success.
	if tag.RowsAffected() == 0 {
		return nil
	}

	// 2) Apply the state change
	tag, err = tx.Exec(ctx,
		`UPDATE trips SET quoted_price = $1 WHERE id = $2`,
		payload.FinalPrice, payload.TripID,
	)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Not found (or already updated in a conflicting way) → rollback so message can be retried
		return fmt.Errorf("trip %s not found", payload.TripID)
	}

	// 3) Commit and we’re done
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
