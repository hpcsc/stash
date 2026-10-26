package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type LinkMetadataFetched struct {
	ItemID      uuid.UUID `json:"itemId"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
}

func (LinkMetadataFetched) EventType() string {
	return domainEvent.Namespace + ".link-metadata-fetched"
}