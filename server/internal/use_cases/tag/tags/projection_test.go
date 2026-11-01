//go:build integration

package tags_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/checkpoint/store"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/subscription"
	"github.com/hpcsc/stash/internal/domain/projection"
	tagEvent "github.com/hpcsc/stash/internal/use_cases/tag/event"
	"github.com/hpcsc/stash/internal/use_cases/tag/tags"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestProjection(t *testing.T) {
	t.Run("keeps two users' tags separate", func(t *testing.T) {
		pool := test.NewDBPool(t)
		clearState(t, pool)
		checkpointStore := store.NewPostgres(pool)
		aliceID, bobID := uuid.New(), uuid.New()
		sub := subscription.NewFake().WithEventChannel(closedChannel(
			tagCreatedEvent("alice", 1, 1, aliceID, "work"),
			tagCreatedEvent("bob", 1, 2, bobID, "work"),
		))

		err := projection.NewProjector(tags.NewProjection(), pool, checkpointStore, sub, test.DiscardLogger()).Start(context.Background())
		require.ErrorIs(t, err, event.ErrSubscriptionEnded)

		alice := tags.NewQuery(pool).For("alice")
		bob := tags.NewQuery(pool).For("bob")

		aliceTags, err := alice.List(context.Background())
		require.NoError(t, err)
		require.Len(t, aliceTags, 1)
		require.Equal(t, "work", aliceTags[0].Name)

		bobTags, err := bob.List(context.Background())
		require.NoError(t, err)
		require.Len(t, bobTags, 1)
		require.Equal(t, "work", bobTags[0].Name)

		aliceSeesBob, err := alice.ByID(context.Background(), bobID.String())
		require.NoError(t, err)
		require.Nil(t, aliceSeesBob)
	})
}

func tagCreatedEvent(user domain.UserID, txID, sequence uint64, id uuid.UUID, name string) domain.Event {
	return domain.Event{
		StreamID: domain.UserStream(user, "tag-"+id.String()),
		Version:  1,
		Position: domain.Position{TransactionID: txID, Sequence: sequence},
		Payload:  &tagEvent.TagCreated{TagID: id, Name: name},
	}
}

func clearState(t *testing.T, pool *pgxpool.Pool) {
	_, err := pool.Exec(context.Background(), "DELETE FROM tags")
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), "DELETE FROM checkpoints WHERE name = 'tags'")
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
