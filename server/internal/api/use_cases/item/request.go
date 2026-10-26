package item

import "github.com/google/uuid"

type CaptureRequest struct {
	Body      string `json:"body" validate:"required"`
	Type      string `json:"type"`
	SourceApp string `json:"sourceApp"`
	SourceURL string `json:"sourceUrl"`
}

type KeepRequest struct {
	Tags []string `json:"tags"`
}

type NoteDocument struct {
	Blocks []NoteDocumentBlock `json:"blocks"`
}

type NoteDocumentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Index *int   `json:"index,omitempty"`
}

func parseTagIDs(raw []string) []uuid.UUID {
	if len(raw) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}
