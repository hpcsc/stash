package item

import "github.com/google/uuid"

type CaptureRequest struct {
	Body      string `json:"body" validate:"required"`
	SourceApp string `json:"sourceApp"`
	SourceURL string `json:"sourceUrl"`
}

type KeepRequest struct {
	Tags []string `json:"tags"`
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
