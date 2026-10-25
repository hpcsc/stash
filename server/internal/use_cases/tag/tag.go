package tag

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/tag/command"
	"github.com/hpcsc/stash/internal/use_cases/tag/event"
)

var ErrAlreadyCreated = errors.New("tag is already created")
var ErrNotCreated = errors.New("tag has not been created")
var ErrNameTaken = errors.New("a tag with that name already exists")

var _ domain.Aggregate = (*Tag)(nil)

func StreamID(id uuid.UUID) string {
	return fmt.Sprintf("tag-%s", id)
}

type Tag struct {
	domain.AggregateBase

	id   uuid.UUID
	name string
}

func (t *Tag) ID() uuid.UUID {
	return t.id
}

func (t *Tag) Name() string {
	return t.name
}

func (t *Tag) Create(cmd command.CreateTag) error {
	if t.id != uuid.Nil {
		return ErrAlreadyCreated
	}
	if cmd.Name == "" {
		return errors.New("name must not be empty")
	}

	t.record(domain.NewEvent(&event.TagCreated{
		TagID: cmd.TagID,
		Name:  cmd.Name,
	}))
	return nil
}

func (t *Tag) Rename(cmd command.RenameTag) error {
	if t.id == uuid.Nil {
		return ErrNotCreated
	}
	if cmd.Name == "" {
		return errors.New("name must not be empty")
	}
	if t.name == cmd.Name {
		return nil
	}

	t.record(domain.NewEvent(&event.TagRenamed{
		TagID: cmd.TagID,
		Name:  cmd.Name,
	}))
	return nil
}

func (t *Tag) Apply(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.TagCreated:
		t.id = p.TagID
		t.name = p.Name
	case *event.TagRenamed:
		t.name = p.Name
	}
}

func (t *Tag) record(e domain.Event) {
	t.Apply(e)
	t.Record(e)
}