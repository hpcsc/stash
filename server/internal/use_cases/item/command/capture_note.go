package command

import (
	"time"

	"github.com/google/uuid"
)

const (
	MaxNoteImages       = 10
	MaxNoteSize   int64 = 100 << 20 // total image bytes in one note
)

type CaptureNote struct {
	ItemID     uuid.UUID
	Markdown   string
	CapturedAt time.Time
}
