package events

import (
	"encoding/json"
	"time"
)

type Position struct {
	TransactionID uint64 `json:"transactionId"`
	Sequence      uint64 `json:"sequence"`
}

type Event struct {
	ID            string          `json:"id"`
	StreamID      string          `json:"streamId"`
	Version       uint64          `json:"version"`
	Position      Position        `json:"position"`
	Type          string          `json:"type"`
	SchemaVersion int             `json:"schemaVersion"`
	CreatedAt     time.Time       `json:"createdAt"`
	Data          json.RawMessage `json:"data"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	Decoded       json.RawMessage `json:"decoded,omitempty"`
	ParseError    string          `json:"parseError,omitempty"`
	CorrelationID string          `json:"correlationId,omitempty"`
	CausationID   string          `json:"causationId,omitempty"`
}

type Page struct {
	Events []Event   `json:"events"`
	Next   *Position `json:"next,omitempty"`
}

type StreamSummary struct {
	StreamID      string    `json:"streamId"`
	EventCount    int64     `json:"eventCount"`
	LastVersion   uint64    `json:"lastVersion"`
	LastCreatedAt time.Time `json:"lastCreatedAt"`
}

type NextEvent struct {
	Position Position `json:"position"`
	Type     string   `json:"type"`
	StreamID string   `json:"streamId"`
}

type Checkpoint struct {
	Name       string     `json:"name"`
	Position   Position   `json:"position"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	Kind       string     `json:"kind,omitempty"`
	EventTypes []string   `json:"eventTypes,omitempty"`
	Lag        uint64     `json:"lag"`
	Next       *NextEvent `json:"next,omitempty"`
}

type Checkpoints struct {
	LastPosition *Position    `json:"lastPosition,omitempty"`
	Checkpoints  []Checkpoint `json:"checkpoints"`
}

type ProjectionTable struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Total   int        `json:"total"`
}
