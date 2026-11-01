//go:build unit

package item_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain/event/stream"
	"github.com/hpcsc/stash/internal/use_cases/item"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	t.Run("scopes streams per user", func(t *testing.T) {
		t.Run("another user's item id resolves to a fresh item", func(t *testing.T) {
			repo := item.NewRepository(stream.NewMemory())
			id := uuid.New()

			alice := repo.For("alice")
			var aliceItem item.Item
			require.NoError(t, aliceItem.CaptureText(itemCommand.CaptureText{
				ItemID:     id,
				Body:       "hello",
				CapturedAt: time.Now(),
			}))
			require.NoError(t, alice.Save(context.Background(), &aliceItem, uuid.New()))

			bob := repo.For("bob")
			bobItem, err := bob.Load(context.Background(), id)
			require.NoError(t, err)
			require.Equal(t, uuid.Nil, bobItem.ID())
			require.ErrorIs(t, bobItem.Keep(itemCommand.Keep{ItemID: id}), item.ErrNotCaptured)

			aliceReloaded, err := alice.Load(context.Background(), id)
			require.NoError(t, err)
			require.Equal(t, id, aliceReloaded.ID())
		})
	})
}
