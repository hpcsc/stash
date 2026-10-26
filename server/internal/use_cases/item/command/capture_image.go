package command

import (
	"time"

	"github.com/google/uuid"
)

type CaptureImage struct {
	ItemID     uuid.UUID
	FileRef    string
	CapturedAt time.Time
}