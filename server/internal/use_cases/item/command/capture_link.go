package command

import (
	"time"

	"github.com/google/uuid"
)

type CaptureLink struct {
	ItemID     uuid.UUID
	URL        string
	CapturedAt time.Time
}