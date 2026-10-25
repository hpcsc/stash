package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ItemDropped struct {
	ItemID uuid.UUID `json:"itemId"`
}

func (ItemDropped) EventType() string {
	return domainEvent.Namespace + ".item-dropped"
}