package events

import "context"

type ctxKey string

const CtxEventID ctxKey = "event_id"

func WithEventID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxEventID, id)
}

func EventIDFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(CtxEventID).(string)
	return v, ok
}
