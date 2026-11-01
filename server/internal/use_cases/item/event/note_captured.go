package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type NoteCaptured struct {
	ItemID     uuid.UUID `json:"itemId"`
	Markdown   string    `json:"markdown"`
	CapturedAt time.Time `json:"capturedAt"`
}

func (NoteCaptured) EventType() string {
	return domainEvent.Namespace + ".note-captured"
}
