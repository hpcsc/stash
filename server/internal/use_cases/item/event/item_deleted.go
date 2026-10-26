package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ItemDeleted struct {
	ItemID uuid.UUID `json:"itemId"`
}

func (ItemDeleted) EventType() string {
	return domainEvent.Namespace + ".item-deleted"
}
