package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type ImageCaptured struct {
	ItemID     uuid.UUID `json:"itemId"`
	FileRef    string    `json:"fileRef"`
	CapturedAt time.Time `json:"capturedAt"`
}

func (ImageCaptured) EventType() string {
	return domainEvent.Namespace + ".image-captured"
}