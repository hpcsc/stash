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
}

var (
	itemID     = uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71")
	capturedAt = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
)

func captureText() command.CaptureText {
	return command.CaptureText{ItemID: itemID, Body: "hello", SourceApp: "chrome", SourceURL: "https://example.com", CapturedAt: capturedAt}
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

func onlyRecorded(t *testing.T, i *item.Item) domain.Event {
	recorded := i.Recorded()
	require.Len(t, recorded, 1)
	return recorded[0]
}
