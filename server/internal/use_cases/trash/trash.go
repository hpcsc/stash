package trash

import (
	"errors"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/trash/command"
	"github.com/hpcsc/stash/internal/use_cases/trash/event"
)

var ErrEmpty = errors.New("trash is empty")

var _ domain.Aggregate = (*Trash)(nil)

type Trash struct {
	domain.AggregateBase
}

func (t *Trash) Empty(cmd command.EmptyTrash) error {
	if len(cmd.ItemIDs) == 0 {
		return ErrEmpty
	}

	t.record(domain.NewEvent(&event.TrashEmptied{
		ItemIDs: cmd.ItemIDs,
	}))
	return nil
}

func (t *Trash) Apply(domain.Event) {}

func (t *Trash) record(e domain.Event) {
	t.Apply(e)
	t.Record(e)
}
