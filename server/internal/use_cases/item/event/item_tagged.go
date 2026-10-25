package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ItemTagged struct {
	ItemID uuid.UUID   `json:"itemId"`
	TagIDs []uuid.UUID `json:"tagIds"`
}

func (ItemTagged) EventType() string {
	return domainEvent.Namespace + ".item-tagged"
}