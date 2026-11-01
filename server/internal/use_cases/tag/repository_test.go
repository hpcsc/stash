//go:build unit

package tag_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain/event/stream"
	"github.com/hpcsc/stash/internal/use_cases/tag"
	tagCommand "github.com/hpcsc/stash/internal/use_cases/tag/command"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	t.Run("scopes streams per user", func(t *testing.T) {
		t.Run("two users can create a tag with the same name", func(t *testing.T) {
			repo := tag.NewRepository(stream.NewMemory())
			id := uuid.New()

			alice := repo.For("alice")
			var aliceTag tag.Tag
			require.NoError(t, aliceTag.Create(tagCommand.CreateTag{TagID: id, Name: "work"}))
			require.NoError(t, alice.Save(context.Background(), &aliceTag, uuid.New()))

			bob := repo.For("bob")
			var bobTag tag.Tag
			require.NoError(t, bobTag.Create(tagCommand.CreateTag{TagID: id, Name: "work"}))
			require.NoError(t, bob.Save(context.Background(), &bobTag, uuid.New()))

			aliceReloaded, err := alice.Load(context.Background(), id)
			require.NoError(t, err)
			require.Equal(t, "work", aliceReloaded.Name())

			bobReloaded, err := bob.Load(context.Background(), id)
			require.NoError(t, err)
			require.Equal(t, "work", bobReloaded.Name())
		})

		t.Run("renaming another user's tag id touches only a fresh tag", func(t *testing.T) {
			repo := tag.NewRepository(stream.NewMemory())
			id := uuid.New()

			alice := repo.For("alice")
			var aliceTag tag.Tag
			require.NoError(t, aliceTag.Create(tagCommand.CreateTag{TagID: id, Name: "work"}))
			require.NoError(t, alice.Save(context.Background(), &aliceTag, uuid.New()))

			bob := repo.For("bob")
			bobTag, err := bob.Load(context.Background(), id)
			require.NoError(t, err)
			require.ErrorIs(t, bobTag.Rename(tagCommand.RenameTag{TagID: id, Name: "personal"}), tag.ErrNotCreated)

			aliceReloaded, err := alice.Load(context.Background(), id)
			require.NoError(t, err)
			require.Equal(t, "work", aliceReloaded.Name())
		})
	})
}
