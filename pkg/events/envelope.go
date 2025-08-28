package events

import "time"

type Envelope[T any] struct {
	Event         string    `json:"event"`
	EventID       string    `json:"event_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	TraceID       string    `json:"trace_id"`
	SchemaVersion int       `json:"schema_version"`
	Payload       T         `json:"payload"`
}
