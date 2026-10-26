package event

import (
	"time"

	"github.com/google/uuid"
	domainEvent "github.com/hpcsc/stash/internal/domain/event"
)

type FileCaptured struct {
	ItemID     uuid.UUID `json:"itemId"`
	FileRef    string    `json:"fileRef"`
	FileName   string    `json:"fileName"`
	FileSize   int64     `json:"fileSize"`
	MimeType   string    `json:"mimeType"`
	CapturedAt time.Time `json:"capturedAt"`
}

func (FileCaptured) EventType() string {
	return domainEvent.Namespace + ".file-captured"
}
