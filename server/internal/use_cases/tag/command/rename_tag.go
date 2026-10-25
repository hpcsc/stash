package command

import "github.com/google/uuid"

type RenameTag struct {
	TagID uuid.UUID
	Name  string
}