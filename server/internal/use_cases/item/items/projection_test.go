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
