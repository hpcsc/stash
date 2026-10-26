package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type TrashEmptied struct {
	ItemIDs []uuid.UUID `json:"itemIds"`
}

func (TrashEmptied) EventType() string {
	return domainEvent.Namespace + ".trash-emptied"
}
