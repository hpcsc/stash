//go:build unit

package emptytrashreactor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/reaction/dispatch"
	"github.com/hpcsc/stash/internal/use_cases/item"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
	itemEvent "github.com/hpcsc/stash/internal/use_cases/item/event"
	"github.com/hpcsc/stash/internal/use_cases/trash/emptytrashreactor"
	"github.com/hpcsc/stash/internal/use_cases/trash/event"
	"github.com/stretchr/testify/require"
)

func TestReaction(t *testing.T) {
	t.Run("handle", func(t *testing.T) {
		t.Run("dispatches one delete for each emptied item", func(t *testing.T) {
			first, second := uuid.New(), uuid.New()
			deletes := &recordingDeletes{}
			reaction := given(t, deletes)

			err := reaction.Handle(context.Background(), trashEmptiedEvent("event-1", first, second))

			require.NoError(t, err)
			require.Len(t, deletes.commands, 2)
			require.Equal(t, first, deletes.commands[0].Payload.ItemID)
			require.Equal(t, second, deletes.commands[1].Payload.ItemID)
		})

		t.Run("uses an ID that depends on the event and item", func(t *testing.T) {
			itemID := uuid.New()
			deletes := &recordingDeletes{}
			reaction := given(t, deletes)
			evt := trashEmptiedEvent("event-1", itemID)

			require.NoError(t, reaction.Handle(context.Background(), evt))

			require.Equal(t, dispatch.CommandIDForTarget(reaction.Name(), evt, itemID.String()), deletes.commands[0].ID)
		})

		t.Run("repeats the same delete IDs when the same event is handled again", func(t *testing.T) {
			itemID := uuid.New()
			deletes := &recordingDeletes{}
			reaction := given(t, deletes)
			evt := trashEmptiedEvent("event-1", itemID)

			require.NoError(t, reaction.Handle(context.Background(), evt))
			require.NoError(t, reaction.Handle(context.Background(), evt))

			require.Len(t, deletes.commands, 2)
			require.Equal(t, deletes.commands[0].ID, deletes.commands[1].ID)
		})

		t.Run("keeps deleting the other items when one delete is already done", func(t *testing.T) {
			first, second := uuid.New(), uuid.New()
			deletes := &recordingDeletes{errByItem: map[uuid.UUID]error{first: item.ErrNotDropped}}
			reaction := given(t, deletes)

			err := reaction.Handle(context.Background(), trashEmptiedEvent("event-1", first, second))

			require.NoError(t, err)
			require.Len(t, deletes.commands, 2)
		})

		t.Run("returns no error when a delete fails", func(t *testing.T) {
			itemID := uuid.New()
			deletes := &recordingDeletes{errByItem: map[uuid.UUID]error{itemID: errors.New("event store unavailable")}}
			reaction := given(t, deletes)

			err := reaction.Handle(context.Background(), trashEmptiedEvent("event-1", itemID))

			require.NoError(t, err)
		})

		t.Run("ignores an event with another payload", func(t *testing.T) {
			deletes := &recordingDeletes{}
			reaction := given(t, deletes)

			err := reaction.Handle(context.Background(), &domain.Event{ID: "event-1", Payload: &itemEvent.ItemDeleted{ItemID: uuid.New()}})

			require.NoError(t, err)
			require.Empty(t, deletes.commands)
		})
	})
}

type recordingDeletes struct {
	commands  []domain.Command[itemCommand.Delete]
	errByItem map[uuid.UUID]error
}

func (h *recordingDeletes) Handle(_ context.Context, cmd domain.Command[itemCommand.Delete]) error {
	h.commands = append(h.commands, cmd)
	return h.errByItem[cmd.Payload.ItemID]
}

func given(t *testing.T, handler *recordingDeletes) *emptytrashreactor.Reaction {
	t.Helper()

	bus := command.NewBus()
	bus.Register[itemCommand.Delete](handler)
	return emptytrashreactor.New(bus, test.DiscardLogger())
}

func trashEmptiedEvent(id string, itemIDs ...uuid.UUID) *domain.Event {
	return &domain.Event{ID: id, Payload: &event.TrashEmptied{ItemIDs: itemIDs}}
}
