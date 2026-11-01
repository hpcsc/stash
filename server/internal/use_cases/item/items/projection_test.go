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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var testUser = domain.UserID("test-user")

func TestProjection(t *testing.T) {
	t.Run("projects a captured note", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			noteCapturedEvent(1, 1, id, "## title\n\nremember this", time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		item, err := items.NewQuery(pool).For(testUser).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, "note", item.Type)
		require.Equal(t, "## title\n\nremember this", item.Body)
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

		item, err := items.NewQuery(pool).For(testUser).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, "file", item.Type)
		require.Equal(t, "files/report.pdf", item.FilePath)
		require.Equal(t, "report.pdf", item.FileName)
		require.Equal(t, int64(1024), item.FileSize)
		require.Equal(t, "application/pdf", item.MimeType)
	})

	t.Run("restores a dropped item to the inbox", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		capturedAt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			noteCapturedEvent(3, 1, id, "restore me", capturedAt),
			itemDroppedEvent(3, 2, 2, id),
			itemRestoredEvent(3, 3, 3, id, "inbox"),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		item, err := items.NewQuery(pool).For(testUser).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, "inbox", item.State)
	})

	t.Run("restores a dropped item that was kept to the library", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		capturedAt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			noteCapturedEvent(4, 1, id, "restore me", capturedAt),
			itemKeptEvent(4, 2, 2, id),
			itemDroppedEvent(4, 3, 3, id),
			itemRestoredEvent(4, 4, 4, id, "library"),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		item, err := items.NewQuery(pool).For(testUser).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.NotNil(t, item)
		require.Equal(t, "kept", item.State)
	})

	t.Run("deletes a dropped item", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		capturedAt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			noteCapturedEvent(5, 1, id, "delete me", capturedAt),
			itemDroppedEvent(5, 2, 2, id),
			itemDeletedEvent(5, 3, 3, id),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		item, err := items.NewQuery(pool).For(testUser).ByID(context.Background(), id.String())
		require.NoError(t, err)
		require.Nil(t, item)
	})

	t.Run("lists items newest first", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			noteCapturedEvent(2, 1, uuid.New(), "older", time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)),
			noteCapturedEvent(2, 2, uuid.New(), "newer", time.Date(2026, 10, 24, 13, 0, 0, 0, time.UTC)),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		all, err := items.NewQuery(pool).For(testUser).List(context.Background(), "")
		require.NoError(t, err)
		require.Len(t, all, 2)
		require.Equal(t, "newer", all[0].Body)
		require.Equal(t, "older", all[1].Body)
	})

	t.Run("finds an existing item by type and body", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		id := uuid.New()
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			noteCapturedEvent(6, 1, id, "remember this", time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		existing, err := items.NewQuery(pool).For(testUser).Existing(context.Background(), "note", "remember this")
		require.NoError(t, err)
		require.NotNil(t, existing)
		require.Equal(t, id.String(), existing.ID)
		require.Equal(t, "inbox", existing.State)

		missing, err := items.NewQuery(pool).For(testUser).Existing(context.Background(), "note", "something else")
		require.NoError(t, err)
		require.Nil(t, missing)
	})

	t.Run("keeps two users' items separate", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		aliceID, bobID := uuid.New(), uuid.New()
		capturedAt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			userNoteCapturedEvent("alice", 7, 1, aliceID, "shared", capturedAt),
			userNoteCapturedEvent("bob", 7, 2, bobID, "shared", capturedAt),
		))

		err := projection.NewProjector(items.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		alice := items.NewQuery(pool).For("alice")
		bob := items.NewQuery(pool).For("bob")

		aliceAll, err := alice.List(context.Background(), "")
		require.NoError(t, err)
		require.Len(t, aliceAll, 1)
		require.Equal(t, aliceID.String(), aliceAll[0].ID)

		bobAll, err := bob.List(context.Background(), "")
		require.NoError(t, err)
		require.Len(t, bobAll, 1)
		require.Equal(t, bobID.String(), bobAll[0].ID)

		aliceSeesBob, err := alice.ByID(context.Background(), bobID.String())
		require.NoError(t, err)
		require.Nil(t, aliceSeesBob)
	})
}

func noteCapturedEvent(txID, sequence uint64, id uuid.UUID, markdown string, capturedAt time.Time) domain.Event {
	return userNoteCapturedEvent(testUser, txID, sequence, id, markdown, capturedAt)
}

func userNoteCapturedEvent(user domain.UserID, txID, sequence uint64, id uuid.UUID, markdown string, capturedAt time.Time) domain.Event {
	return domain.Event{
		StreamID: domain.UserStream(user, "item-"+id.String()),
		Version:  1,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload: &itemEvent.NoteCaptured{
			ItemID:     id,
			Markdown:   markdown,
			CapturedAt: capturedAt,
		},
	}
}

func fileCapturedEvent(txID, sequence uint64, id uuid.UUID, capturedAt time.Time) domain.Event {
	return domain.Event{
		StreamID: domain.UserStream(testUser, "item-"+id.String()),
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

func itemKeptEvent(txID, sequence, version uint64, id uuid.UUID) domain.Event {
	return domain.Event{
		StreamID: domain.UserStream(testUser, "item-"+id.String()),
		Version:  version,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload:  &itemEvent.ItemKept{ItemID: id},
	}
}

func itemDroppedEvent(txID, sequence, version uint64, id uuid.UUID) domain.Event {
	return domain.Event{
		StreamID: domain.UserStream(testUser, "item-"+id.String()),
		Version:  version,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload:  &itemEvent.ItemDropped{ItemID: id},
	}
}

func itemRestoredEvent(txID, sequence, version uint64, id uuid.UUID, restoredTo string) domain.Event {
	return domain.Event{
		StreamID: domain.UserStream(testUser, "item-"+id.String()),
		Version:  version,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload:  &itemEvent.ItemRestored{ItemID: id, RestoredTo: restoredTo},
	}
}

func itemDeletedEvent(txID, sequence, version uint64, id uuid.UUID) domain.Event {
	return domain.Event{
		StreamID: domain.UserStream(testUser, "item-"+id.String()),
		Version:  version,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload:  &itemEvent.ItemDeleted{ItemID: id},
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
