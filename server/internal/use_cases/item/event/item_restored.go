package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ItemRestored struct {
	ItemID     uuid.UUID `json:"itemId"`
	RestoredTo string    `json:"restoredTo"`
}

func (ItemRestored) EventType() string {
	return domainEvent.Namespace + ".item-restored"
}
