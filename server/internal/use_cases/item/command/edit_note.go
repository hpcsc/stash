package command

import "github.com/google/uuid"

type EditNote struct {
	ItemID          uuid.UUID
	Markdown        string
	ExpectedVersion uint64
}