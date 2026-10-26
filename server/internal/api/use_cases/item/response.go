package item

import (
	"time"

	"github.com/hpcsc/stash/internal/use_cases/item/items"
)

type NoteResponse struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	Body       string        `json:"body"`
	State      string        `json:"state"`
	CapturedAt time.Time     `json:"capturedAt"`
	Images     []items.Image `json:"images,omitempty"`
	Blocks     []items.Block `json:"blocks,omitempty"`
}
