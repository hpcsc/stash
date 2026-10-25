//go:build unit

package item_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/item"
	"github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
	"github.com/stretchr/testify/require"
)

func TestItem(t *testing.T) {
	t.Run("capture text", func(t *testing.T) {
		t.Run("captures text into the inbox", func(t *testing.T) {
			var i item.Item
			cmd := captureText()

			require.NoError(t, i.CaptureText(cmd))

			require.Equal(t, &event.TextCaptured{ItemID: cmd.ItemID, Body: "hello", SourceApp: "chrome", SourceURL: "https://example.com", CapturedAt: capturedAt}, onlyRecorded(t, &i).Payload)
			require.Equal(t, cmd.ItemID, i.ID())
		})

		t.Run("rejects a second capture", func(t *testing.T) {
			i := given(captured())

			require.ErrorIs(t, i.CaptureText(captureText()), item.ErrAlreadyCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects an empty body", func(t *testing.T) {
			var i item.Item
			cmd := captureText()
			cmd.Body = ""

			require.Error(t, i.CaptureText(cmd))
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("keep", func(t *testing.T) {
		t.Run("keeps an inbox item", func(t *testing.T) {
			i := given(captured())

			require.NoError(t, i.Keep(keepCmd()))

			require.Equal(t, &event.ItemKept{ItemID: itemID}, onlyRecorded(t, i).Payload)
		})

		t.Run("rejects keeping an item that is not captured", func(t *testing.T) {
			var i item.Item

			require.ErrorIs(t, i.Keep(keepCmd()), item.ErrNotCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects keeping an already-kept item", func(t *testing.T) {
			i := given(captured(), kept())

			require.ErrorIs(t, i.Keep(keepCmd()), item.ErrAlreadyKept)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects keeping a dropped item", func(t *testing.T) {
			i := given(captured(), dropped())

			require.ErrorIs(t, i.Keep(keepCmd()), item.ErrAlreadyDropped)
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("drop", func(t *testing.T) {
		t.Run("drops an inbox item", func(t *testing.T) {
			i := given(captured())

			require.NoError(t, i.Drop(dropCmd()))

			require.Equal(t, &event.ItemDropped{ItemID: itemID}, onlyRecorded(t, i).Payload)
		})

		t.Run("rejects dropping an item that is not captured", func(t *testing.T) {
			var i item.Item

			require.ErrorIs(t, i.Drop(dropCmd()), item.ErrNotCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects dropping an already-dropped item", func(t *testing.T) {
			i := given(captured(), dropped())

			require.ErrorIs(t, i.Drop(dropCmd()), item.ErrAlreadyDropped)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects dropping a kept item", func(t *testing.T) {
			i := given(captured(), kept())

			require.ErrorIs(t, i.Drop(dropCmd()), item.ErrAlreadyKept)
			require.Empty(t, i.Recorded())
		})
	})
}

var (
	itemID     = uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71")
	capturedAt = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
)

func captureText() command.CaptureText {
	return command.CaptureText{ItemID: itemID, Body: "hello", SourceApp: "chrome", SourceURL: "https://example.com", CapturedAt: capturedAt}
}

func keepCmd() command.Keep {
	return command.Keep{ItemID: itemID}
}

func dropCmd() command.Drop {
	return command.Drop{ItemID: itemID}
}

func given(history ...domain.Event) *item.Item {
	var i item.Item
	for _, e := range history {
		i.Apply(e)
	}
	i.MarkCommitted(uint64(len(history)))
	return &i
}

func captured() domain.Event {
	return domain.NewEvent(&event.TextCaptured{ItemID: itemID, Body: "hello", CapturedAt: capturedAt})
}

func kept() domain.Event {
	return domain.NewEvent(&event.ItemKept{ItemID: itemID})
}

func dropped() domain.Event {
	return domain.NewEvent(&event.ItemDropped{ItemID: itemID})
}

func onlyRecorded(t *testing.T, i *item.Item) domain.Event {
	recorded := i.Recorded()
	require.Len(t, recorded, 1)
	return recorded[0]
}
