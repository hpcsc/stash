package command

import (
	"time"

	"github.com/google/uuid"
)

const MaxFileSize int64 = 25 << 20 // 25 MB

type CaptureFile struct {
	ItemID     uuid.UUID
	FileRef    string
	FileName   string
	FileSize   int64
	MimeType   string
	CapturedAt time.Time
}
