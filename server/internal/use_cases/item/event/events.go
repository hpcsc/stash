package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type TextCaptured struct {
	ItemID     uuid.UUID `json:"itemId"`
	Body       string    `json:"body"`
	SourceApp  string    `json:"sourceApp,omitempty"`
	SourceURL  string    `json:"sourceUrl,omitempty"`
	CapturedAt time.Time `json:"capturedAt"`
}

func (TextCaptured) EventType() string {
	return domainEvent.Namespace + ".text-captured"
}

type ItemKept struct {
	ItemID uuid.UUID `json:"itemId"`
}

func (ItemKept) EventType() string {
	return domainEvent.Namespace + ".item-kept"
}

type ItemDropped struct {
	ItemID uuid.UUID `json:"itemId"`
}

func (ItemDropped) EventType() string {
	return domainEvent.Namespace + ".item-dropped"
}
