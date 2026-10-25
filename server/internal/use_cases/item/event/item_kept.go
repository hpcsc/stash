package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ItemKept struct {
	ItemID uuid.UUID `json:"itemId"`
}

func (ItemKept) EventType() string {
	return domainEvent.Namespace + ".item-kept"
}