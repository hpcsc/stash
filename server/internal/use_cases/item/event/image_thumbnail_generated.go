package event

import (
	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ImageThumbnailGenerated struct {
	ItemID       uuid.UUID `json:"itemId"`
	ImageID      uuid.UUID `json:"imageId"`
	ThumbnailRef string    `json:"thumbnailRef"`
}

func (ImageThumbnailGenerated) EventType() string {
	return domainEvent.Namespace + ".image-thumbnail-generated"
}
