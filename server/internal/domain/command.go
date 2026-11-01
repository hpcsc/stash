package domain

import "github.com/google/uuid"

type Command[P any] struct {
	ID      uuid.UUID
	Actor   UserID
	Payload P
}

func NewCommand[P any](actor UserID, payload P) Command[P] {
	return Command[P]{
		ID:      uuid.New(),
		Actor:   actor,
		Payload: payload,
	}
}
