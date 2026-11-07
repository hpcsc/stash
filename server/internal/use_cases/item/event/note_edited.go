package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type NoteEdited struct {
	ItemID   uuid.UUID `json:"itemId"`
	Markdown string    `json:"markdown"`
}

func (NoteEdited) EventType() string {
	return domainEvent.Namespace + ".note-edited"
}