package item

import "time"

type CaptureResponse struct {
	ID             string `json:"id"`
	Duplicate      bool   `json:"duplicate"`
	DuplicateState string `json:"duplicateState,omitempty"`
}

type NoteResponse struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Body       string    `json:"body"`
	State      string    `json:"state"`
	CapturedAt time.Time `json:"capturedAt"`
}
