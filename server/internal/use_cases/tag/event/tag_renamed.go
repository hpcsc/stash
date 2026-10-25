package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type TagRenamed struct {
	TagID uuid.UUID `json:"tagId"`
	Name  string    `json:"name"`
}

func (TagRenamed) EventType() string {
	return domainEvent.Namespace + ".tag-renamed"
}