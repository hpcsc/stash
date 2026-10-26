package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ThumbnailGenerated struct {
	ItemID       uuid.UUID `json:"itemId"`
	ThumbnailRef string    `json:"thumbnailRef"`
}

func (ThumbnailGenerated) EventType() string {
	return domainEvent.Namespace + ".thumbnail-generated"
}