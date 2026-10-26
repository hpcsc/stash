//go:build unit

package trash_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/use_cases/trash"
	"github.com/hpcsc/stash/internal/use_cases/trash/command"
	"github.com/hpcsc/stash/internal/use_cases/trash/event"
	"github.com/stretchr/testify/require"
)

func TestTrash(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		t.Run("empties the trash with the given items", func(t *testing.T) {
			var trashAggregate trash.Trash
			itemIDs := []uuid.UUID{uuid.New(), uuid.New()}

			require.NoError(t, trashAggregate.Empty(command.EmptyTrash{ItemIDs: itemIDs}))

			recorded := trashAggregate.Recorded()
			require.Len(t, recorded, 1)
			require.Equal(t, &event.TrashEmptied{ItemIDs: itemIDs}, recorded[0].Payload)
		})

		t.Run("rejects emptying an empty trash", func(t *testing.T) {
			var trashAggregate trash.Trash

			require.ErrorIs(t, trashAggregate.Empty(command.EmptyTrash{}), trash.ErrEmpty)
			require.Empty(t, trashAggregate.Recorded())
		})
	})
}
