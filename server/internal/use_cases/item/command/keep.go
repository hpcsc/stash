package command

import "github.com/google/uuid"

type Keep struct {
	ItemID uuid.UUID
	TagIDs []uuid.UUID
}