package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	ev "github.com/sezarsaman/intercity-sim/pkg/events"
)

type PricedHandler struct {
	DB *pgxpool.Pool
}

func NewPricedHandler(db *pgxpool.Pool) *PricedHandler { return &PricedHandler{DB: db} }

func (h *PricedHandler) Handle(ctx context.Context, e ev.TripPriced) error {
	_, err := h.DB.Exec(ctx, "UPDATE trips SET quoted_price=$1 WHERE id=$2", e.FinalPrice, e.TripID)
	return err
}
