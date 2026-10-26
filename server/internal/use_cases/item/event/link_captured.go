package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type LinkCaptured struct {
	ItemID     uuid.UUID `json:"itemId"`
	URL        string    `json:"url"`
	CapturedAt time.Time `json:"capturedAt"`
}

func (LinkCaptured) EventType() string {
	return domainEvent.Namespace + ".link-captured"
}