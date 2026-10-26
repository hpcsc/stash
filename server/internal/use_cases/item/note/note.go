package note

import (
	"strings"

	"github.com/google/uuid"
)

const (
	TypeText  = "text"
	TypeImage = "image"
)

type Block struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Image *Image `json:"image,omitempty"`
}

type Image struct {
	ID       uuid.UUID `json:"id"`
	FileRef  string    `json:"fileRef"`
	FileName string    `json:"fileName,omitempty"`
	FileSize int64     `json:"fileSize,omitempty"`
	MimeType string    `json:"mimeType,omitempty"`
}

func PlainText(blocks []Block) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == TypeText && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, strings.TrimSpace(b.Text))
		}
	}
	return strings.Join(parts, "\n")
}

func HasText(blocks []Block) bool {
	for _, b := range blocks {
		if b.Type == TypeText && strings.TrimSpace(b.Text) != "" {
			return true
		}
	}
	return false
}

func HasImages(blocks []Block) bool {
	for _, b := range blocks {
		if b.Type == TypeImage && b.Image != nil {
			return true
		}
	}
	return false
}
