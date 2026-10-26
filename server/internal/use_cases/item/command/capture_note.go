package command

import (
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/use_cases/item/note"
)

const (
	MaxNoteImages       = 10
	MaxNoteSize   int64 = 100 << 20 // total image bytes in one note
)

type CaptureNote struct {
	ItemID     uuid.UUID
	Blocks     []note.Block
	CapturedAt time.Time
}
