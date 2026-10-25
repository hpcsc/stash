package command

import "github.com/google/uuid"

type CreateTag struct {
	TagID uuid.UUID
	Name  string
}