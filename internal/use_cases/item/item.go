package item

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
)

var ErrAlreadyCaptured = errors.New("item is already captured")

var _ domain.Aggregate = (*Item)(nil)

func StreamID(id uuid.UUID) string {
	return fmt.Sprintf("item-%s", id)
}

type Item struct {
	domain.AggregateBase

	id uuid.UUID
}

func (i *Item) ID() uuid.UUID {
	return i.id
}

func (i *Item) CaptureText(cmd command.CaptureText) error {
	if i.id != uuid.Nil {
		return ErrAlreadyCaptured
	}
	if cmd.Body == "" {
		return errors.New("body must not be empty")
	}

	i.record(domain.NewEvent(&event.TextCaptured{
		ItemID:     cmd.ItemID,
		Body:       cmd.Body,
		SourceApp:  cmd.SourceApp,
		SourceURL:  cmd.SourceURL,
		CapturedAt: cmd.CapturedAt,
	}))
	return nil
}

func (i *Item) Apply(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.TextCaptured:
		i.id = p.ItemID
	}
}

func (i *Item) record(e domain.Event) {
	i.Apply(e)
	i.Record(e)
}
