package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/use_cases/item/note"
)

type NoteCaptured struct {
	ItemID     uuid.UUID    `json:"itemId"`
	Blocks     []note.Block `json:"blocks"`
	CapturedAt time.Time    `json:"capturedAt"`
}

func (NoteCaptured) EventType() string {
	return domainEvent.Namespace + ".note-captured"
}
