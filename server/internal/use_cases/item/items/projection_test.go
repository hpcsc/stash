//go:build integration

package items_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/checkpoint/store"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/subscription"
	"github.com/hpcsc/stash/internal/domain/projection"
	itemEvent "github.com/hpcsc/stash/internal/use_cases/item/event"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	"github.com/hpcsc/stash/internal/use_cases/item/note"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestProjection(t *testing.T) {
	t.Run("projects a captured text item", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			textCapturedEvent(1, 1, id, "remember this", "chrome", time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		item, err := items.NewQuery(pool).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, "text", item.Type)
		require.Equal(t, "remember this", item.Body)
		require.Equal(t, "chrome", item.SourceApp)
	})

	t.Run("projects a captured file with its metadata", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			fileCapturedEvent(1, 1, id, time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		item, err := items.NewQuery(pool).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, "file", item.Type)
		require.Equal(t, "files/report.pdf", item.FilePath)
		require.Equal(t, "report.pdf", item.FileName)
		require.Equal(t, int64(1024), item.FileSize)
		require.Equal(t, "application/pdf", item.MimeType)
	})

	t.Run("projects a note with its blocks and images", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		imageID := uuid.New()
		capturedAt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			noteCapturedEvent(1, 1, id, imageID, capturedAt),
			imageThumbnailGeneratedEvent(1, 2, id, imageID),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		item, err := items.NewQuery(pool).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, "note", item.Type)
		require.Equal(t, "before\nafter", item.Body)
		require.Len(t, item.Images, 1)
		require.Equal(t, imageID.String(), item.Images[0].ID)
		require.Equal(t, "images/photo.png", item.Images[0].FilePath)
		require.Equal(t, "thumbnails/photo.png", item.Images[0].ThumbnailPath)
		require.Len(t, item.Blocks, 3)
		require.Equal(t, "before", item.Blocks[0].Text)
		require.Equal(t, imageID.String(), item.Blocks[1].Image.ID)
		require.Equal(t, "after", item.Blocks[2].Text)
	})

	t.Run("lists items newest first", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			textCapturedEvent(2, 1, uuid.New(), "older", "", time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)),
			textCapturedEvent(2, 2, uuid.New(), "newer", "", time.Date(2026, 10, 24, 13, 0, 0, 0, time.UTC)),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		all, err := items.NewQuery(pool).List(context.Background(), "")
		require.NoError(t, err)
		require.Len(t, all, 2)
		require.Equal(t, "newer", all[0].Body)
		require.Equal(t, "older", all[1].Body)
	})
}

func textCapturedEvent(txID, sequence uint64, id uuid.UUID, body, sourceApp string, capturedAt time.Time) domain.Event {
	return domain.Event{
		StreamID: "item-" + id.String(),
		Version:  1,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload: &itemEvent.TextCaptured{
			ItemID:     id,
			Body:       body,
			SourceApp:  sourceApp,
			CapturedAt: capturedAt,
		},
	}
}

func fileCapturedEvent(txID, sequence uint64, id uuid.UUID, capturedAt time.Time) domain.Event {
	return domain.Event{
		StreamID: "item-" + id.String(),
		Version:  1,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload: &itemEvent.FileCaptured{
			ItemID:     id,
			FileRef:    "files/report.pdf",
			FileName:   "report.pdf",
			FileSize:   1024,
			MimeType:   "application/pdf",
			CapturedAt: capturedAt,
		},
	}
}

func noteCapturedEvent(txID, sequence uint64, id, imageID uuid.UUID, capturedAt time.Time) domain.Event {
	return domain.Event{
		StreamID: "item-" + id.String(),
		Version:  1,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload: &itemEvent.NoteCaptured{
			ItemID: id,
			Blocks: []note.Block{
				{Type: note.TypeText, Text: "before"},
				{Type: note.TypeImage, Image: &note.Image{
					ID:       imageID,
					FileRef:  "images/photo.png",
					FileName: "photo.png",
					FileSize: 1024,
					MimeType: "image/png",
				}},
				{Type: note.TypeText, Text: "after"},
			},
			CapturedAt: capturedAt,
		},
	}
}

func imageThumbnailGeneratedEvent(txID, sequence uint64, id, imageID uuid.UUID) domain.Event {
	return domain.Event{
		StreamID: "item-" + id.String(),
		Version:  2,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload: &itemEvent.ImageThumbnailGenerated{
			ItemID:       id,
			ImageID:      imageID,
			ThumbnailRef: "thumbnails/photo.png",
		},
	}
}

func clearState(t *testing.T, pool *pgxpool.Pool) {
	_, err := pool.Exec(context.Background(), "DELETE FROM items")
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), "DELETE FROM checkpoints WHERE name = 'items'")
	require.NoError(t, err)
}

func closedChannel(events ...domain.Event) <-chan domain.Event {
	ch := make(chan domain.Event, len(events))
	for _, e := range events {
		ch <- e
	}
	close(ch)
	return ch
}
