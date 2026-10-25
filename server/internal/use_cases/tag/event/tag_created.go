package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type TagCreated struct {
	TagID uuid.UUID `json:"tagId"`
	Name  string    `json:"name"`
}

func (TagCreated) EventType() string {
	return domainEvent.Namespace + ".tag-created"
}