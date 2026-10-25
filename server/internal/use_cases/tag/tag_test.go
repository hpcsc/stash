//go:build unit

package tag_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/tag"
	"github.com/hpcsc/stash/internal/use_cases/tag/command"
	"github.com/hpcsc/stash/internal/use_cases/tag/event"
	"github.com/stretchr/testify/require"
)

func TestTag(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		t.Run("creates a tag", func(t *testing.T) {
			var tg tag.Tag
			cmd := createCmd()

			require.NoError(t, tg.Create(cmd))

			require.Equal(t, &event.TagCreated{TagID: tagID, Name: "work"}, onlyRecorded(t, &tg).Payload)
			require.Equal(t, tagID, tg.ID())
			require.Equal(t, "work", tg.Name())
		})

		t.Run("rejects a second create", func(t *testing.T) {
			tg := given(created())

			require.ErrorIs(t, tg.Create(createCmd()), tag.ErrAlreadyCreated)
			require.Empty(t, tg.Recorded())
		})

		t.Run("rejects an empty name", func(t *testing.T) {
			var tg tag.Tag
			cmd := createCmd()
			cmd.Name = ""

			require.Error(t, tg.Create(cmd))
			require.Empty(t, tg.Recorded())
		})
	})

	t.Run("rename", func(t *testing.T) {
		t.Run("renames a tag", func(t *testing.T) {
			tg := given(created())

			require.NoError(t, tg.Rename(renameCmd("personal")))

			require.Equal(t, &event.TagRenamed{TagID: tagID, Name: "personal"}, onlyRecorded(t, tg).Payload)
		})

		t.Run("no-ops when name is unchanged", func(t *testing.T) {
			tg := given(created())

			require.NoError(t, tg.Rename(renameCmd("work")))
			require.Empty(t, tg.Recorded())
		})

		t.Run("rejects renaming a tag that does not exist", func(t *testing.T) {
			var tg tag.Tag

			require.ErrorIs(t, tg.Rename(renameCmd("work")), tag.ErrNotCreated)
			require.Empty(t, tg.Recorded())
		})

		t.Run("rejects an empty name", func(t *testing.T) {
			tg := given(created())
			cmd := renameCmd("")
			cmd.Name = ""

			require.Error(t, tg.Rename(cmd))
			require.Empty(t, tg.Recorded())
		})
	})
}

var tagID = uuid.MustParse("8a3f1b2c-6d4e-5f7a-9b01-3c2d8e0f4a61")

func createCmd() command.CreateTag {
	return command.CreateTag{TagID: tagID, Name: "work"}
}

func renameCmd(name string) command.RenameTag {
	return command.RenameTag{TagID: tagID, Name: name}
}

func given(history ...domain.Event) *tag.Tag {
	var t tag.Tag
	for _, e := range history {
		t.Apply(e)
	}
	t.MarkCommitted(uint64(len(history)))
	return &t
}

func created() domain.Event {
	return domain.NewEvent(&event.TagCreated{TagID: tagID, Name: "work"})
}

func onlyRecorded(t *testing.T, tg *tag.Tag) domain.Event {
	recorded := tg.Recorded()
	require.Len(t, recorded, 1)
	return recorded[0]
}