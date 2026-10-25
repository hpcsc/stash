package command

import (
	"time"

	"github.com/google/uuid"
)

type CaptureText struct {
	ItemID     uuid.UUID
	Body       string
	SourceApp  string
	SourceURL  string
	CapturedAt time.Time
}

type Keep struct {
	ItemID uuid.UUID
}

type Drop struct {
	ItemID uuid.UUID
}
