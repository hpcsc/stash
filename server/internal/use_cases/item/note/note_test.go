//go:build unit

package note_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/use_cases/item/note"
	"github.com/stretchr/testify/require"
)

func TestNote(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		t.Run("joins text blocks and ignores images", func(t *testing.T) {
			blocks := []note.Block{
				{Type: note.TypeText, Text: " before "},
				{Type: note.TypeImage, Image: &note.Image{ID: uuid.New(), FileRef: "images/a.png"}},
				{Type: note.TypeText, Text: "after"},
			}

			require.Equal(t, "before\nafter", note.PlainText(blocks))
		})

		t.Run("returns an empty string when there is no text", func(t *testing.T) {
			blocks := []note.Block{{Type: note.TypeImage, Image: &note.Image{ID: uuid.New(), FileRef: "images/a.png"}}}

			require.Empty(t, note.PlainText(blocks))
		})
	})

	t.Run("has text", func(t *testing.T) {
		require.True(t, note.HasText([]note.Block{{Type: note.TypeText, Text: "hello"}}))
		require.False(t, note.HasText([]note.Block{{Type: note.TypeText, Text: "   "}}))
	})

	t.Run("has images", func(t *testing.T) {
		require.True(t, note.HasImages([]note.Block{{Type: note.TypeImage, Image: &note.Image{ID: uuid.New(), FileRef: "images/a.png"}}}))
		require.False(t, note.HasImages([]note.Block{{Type: note.TypeImage}}))
	})
}
