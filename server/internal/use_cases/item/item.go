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
var ErrAlreadyKept = errors.New("item is already kept")
var ErrAlreadyDropped = errors.New("item is already dropped")
var ErrNotCaptured = errors.New("item has not been captured")

var _ domain.Aggregate = (*Item)(nil)

func StreamID(id uuid.UUID) string {
	return fmt.Sprintf("item-%s", id)
}

type Item struct {
	domain.AggregateBase

	id    uuid.UUID
	state string
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

func (i *Item) Keep(cmd command.Keep) error {
	if i.id == uuid.Nil {
		return ErrNotCaptured
	}
	if i.state == "kept" {
		return ErrAlreadyKept
	}
	if i.state == "dropped" {
		return ErrAlreadyDropped
	}

	i.record(domain.NewEvent(&event.ItemKept{
		ItemID: cmd.ItemID,
	}))
	return nil
}

func (i *Item) Drop(cmd command.Drop) error {
	if i.id == uuid.Nil {
		return ErrNotCaptured
	}
	if i.state == "dropped" {
		return ErrAlreadyDropped
	}
	if i.state == "kept" {
		return ErrAlreadyKept
	}

	i.record(domain.NewEvent(&event.ItemDropped{
		ItemID: cmd.ItemID,
	}))
	return nil
}

func (i *Item) Apply(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.TextCaptured:
		i.id = p.ItemID
		i.state = "inbox"
	case *event.ItemKept:
		i.state = "kept"
	case *event.ItemDropped:
		i.state = "dropped"
	}
}

func (i *Item) record(e domain.Event) {
	i.Apply(e)
	i.Record(e)
}
