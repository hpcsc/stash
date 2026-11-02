//go:build integration

package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/api/use_cases/events"
	"github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/checkpoint/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		t.Run("pages events in commit order", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			streamID := "test-list-" + test.NewStringID(t)
			for i := uint64(1); i <= 5; i++ {
				insertEvent(t, pool, streamID, typeName, i, `{}`, nil)
			}

			query := events.NewQuery(pool).Scoped(events.AllScope())
			filter := events.ListFilter{Limit: 2, Types: []string{typeName}}

			first, next, err := query.List(context.Background(), filter)
			require.NoError(t, err)
			require.Len(t, first, 2)
			require.Equal(t, uint64(1), first[0].Version)
			require.Equal(t, uint64(2), first[1].Version)
			require.NotNil(t, next)

			filter.After = next
			second, next, err := query.List(context.Background(), filter)
			require.NoError(t, err)
			require.Len(t, second, 2)
			require.Equal(t, uint64(3), second[0].Version)
			require.Equal(t, uint64(4), second[1].Version)
			require.NotNil(t, next)

			filter.After = next
			third, next, err := query.List(context.Background(), filter)
			require.NoError(t, err)
			require.Len(t, third, 1)
			require.Equal(t, uint64(5), third[0].Version)
			require.Nil(t, next)
		})

		t.Run("scopes a user to their own streams", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			unique := test.NewStringID(t)
			alice := "alice-" + unique
			bob := "bob-" + unique

			insertEvent(t, pool, "user/"+alice+"/item-1", typeName, 1, `{}`, nil)
			insertEvent(t, pool, "identity-"+alice, typeName, 1, `{}`, nil)
			insertEvent(t, pool, "user/"+bob+"/item-1", typeName, 1, `{}`, nil)

			filter := events.ListFilter{Types: []string{typeName}}

			aliceEvents, _, err := events.NewQuery(pool).Scoped(events.UserScope(domain.UserID(alice))).List(context.Background(), filter)
			require.NoError(t, err)
			require.Len(t, aliceEvents, 2)

			bobEvents, _, err := events.NewQuery(pool).Scoped(events.UserScope(domain.UserID(bob))).List(context.Background(), filter)
			require.NoError(t, err)
			require.Len(t, bobEvents, 1)
			require.Equal(t, "user/"+bob+"/item-1", bobEvents[0].StreamID)
		})

		t.Run("filters by correlation id", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			streamID := "test-correlation-" + test.NewStringID(t)
			correlation := uuid.NewString()

			insertEvent(t, pool, streamID, typeName, 1, `{}`, metadata(correlation))
			insertEvent(t, pool, streamID, typeName, 2, `{}`, metadata(uuid.NewString()))

			events, _, err := events.NewQuery(pool).Scoped(events.AllScope()).List(context.Background(), events.ListFilter{
				Types:         []string{typeName},
				CorrelationID: correlation,
			})
			require.NoError(t, err)
			require.Len(t, events, 1)
			require.Equal(t, uint64(1), events[0].Version)
		})

		t.Run("filters by stream prefix", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			unique := test.NewStringID(t)
			user := "user-" + unique

			insertEvent(t, pool, "user/"+user+"/item-1", typeName, 1, `{}`, nil)
			insertEvent(t, pool, "user/"+user+"/tag-1", typeName, 1, `{}`, nil)

			events, _, err := events.NewQuery(pool).Scoped(events.AllScope()).List(context.Background(), events.ListFilter{
				Types:        []string{typeName},
				StreamPrefix: "user/" + user + "/item",
			})
			require.NoError(t, err)
			require.Len(t, events, 1)
			require.Equal(t, "user/"+user+"/item-1", events[0].StreamID)
		})
	})

	t.Run("stream", func(t *testing.T) {
		t.Run("returns a stream's events in version order", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			streamID := "test-stream-" + test.NewStringID(t)
			insertEvent(t, pool, streamID, typeName, 3, `{}`, nil)
			insertEvent(t, pool, streamID, typeName, 1, `{}`, nil)
			insertEvent(t, pool, streamID, typeName, 2, `{}`, nil)

			events, err := events.NewQuery(pool).Scoped(events.AllScope()).Stream(context.Background(), streamID)
			require.NoError(t, err)
			require.Len(t, events, 3)
			require.Equal(t, uint64(1), events[0].Version)
			require.Equal(t, uint64(2), events[1].Version)
			require.Equal(t, uint64(3), events[2].Version)
		})

		t.Run("hides another user's stream", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			streamID := "user/bob-" + test.NewStringID(t) + "/item-1"
			insertEvent(t, pool, streamID, typeName, 1, `{}`, nil)

			events, err := events.NewQuery(pool).Scoped(events.UserScope("alice")).Stream(context.Background(), streamID)
			require.NoError(t, err)
			require.Empty(t, events)
		})
	})

	t.Run("streams", func(t *testing.T) {
		t.Run("summarizes each stream for the scope", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			unique := test.NewStringID(t)
			alice := "alice-" + unique
			bob := "bob-" + unique

			insertEvent(t, pool, "user/"+alice+"/item-1", typeName, 1, `{}`, nil)
			insertEvent(t, pool, "user/"+alice+"/item-1", typeName, 2, `{}`, nil)
			insertEvent(t, pool, "user/"+bob+"/item-1", typeName, 1, `{}`, nil)

			streams, err := events.NewQuery(pool).Scoped(events.UserScope(domain.UserID(alice))).Streams(context.Background())
			require.NoError(t, err)
			require.Len(t, streams, 1)
			require.Equal(t, "user/"+alice+"/item-1", streams[0].StreamID)
			require.Equal(t, int64(2), streams[0].EventCount)
			require.Equal(t, uint64(2), streams[0].LastVersion)
		})
	})

	t.Run("types", func(t *testing.T) {
		t.Run("returns the distinct types for the scope", func(t *testing.T) {
			pool := test.NewDBPool(t)
			first := uniqueType(t)
			second := uniqueType(t)
			streamID := "test-types-" + test.NewStringID(t)
			insertEvent(t, pool, streamID, first, 1, `{}`, nil)
			insertEvent(t, pool, streamID, second, 2, `{}`, nil)

			types, err := events.NewQuery(pool).Scoped(events.AllScope()).Types(context.Background())
			require.NoError(t, err)
			require.Contains(t, types, first)
			require.Contains(t, types, second)
		})
	})

	t.Run("checkpoints", func(t *testing.T) {
		t.Run("returns each checkpoint with its lag", func(t *testing.T) {
			pool := test.NewDBPool(t)
			name := "checkpoints-" + test.NewStringID(t)
			require.NoError(t, store.NewPostgres(pool).Set(context.Background(), name, domain.Position{}))
			insertEvent(t, pool, "test-checkpoint-"+test.NewStringID(t), uniqueType(t), 1, `{}`, nil)

			checkpoints, head, err := events.NewQuery(pool).Scoped(events.AllScope()).Checkpoints(context.Background())
			require.NoError(t, err)
			require.NotNil(t, head)

			var found events.CheckpointRow
			for _, c := range checkpoints {
				if c.Name == name {
					found = c
				}
			}
			require.Equal(t, name, found.Name)
			require.Equal(t, domain.Position{}, found.Position)
		})
	})

	t.Run("tail", func(t *testing.T) {
		t.Run("emits events after the given position", func(t *testing.T) {
			pool := test.NewDBPool(t)
			typeName := uniqueType(t)
			streamID := "test-tail-" + test.NewStringID(t)
			insertEvent(t, pool, streamID, typeName, 1, `{}`, nil)
			insertEvent(t, pool, streamID, typeName, 2, `{}`, nil)
			insertEvent(t, pool, streamID, typeName, 3, `{}`, nil)

			all, _, err := events.NewQuery(pool).Scoped(events.AllScope()).List(context.Background(), events.ListFilter{Types: []string{typeName}})
			require.NoError(t, err)
			require.Len(t, all, 3)

			after := &domain.Position{TransactionID: all[1].Position.TransactionID, Sequence: all[1].Position.Sequence}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			emitted := make(chan events.StoredEvent, 1)
			go func() {
				_ = events.NewQuery(pool).Scoped(events.AllScope()).Tail(ctx, after, func(e events.StoredEvent) error {
					emitted <- e
					return nil
				})
			}()

			select {
			case event := <-emitted:
				require.Equal(t, uint64(3), event.Version)
			case <-time.After(2 * time.Second):
				require.Fail(t, "no event was emitted")
			}
		})
	})
}

func uniqueType(t *testing.T) string {
	t.Helper()
	return "test." + test.NewStringID(t)
}

func metadata(correlationID string) *string {
	value := `{"CorrelationID":"` + correlationID + `","CausationID":"` + correlationID + `"}`
	return &value
}

func insertEvent(t *testing.T, pool *pgxpool.Pool, streamID, typeName string, version uint64, data string, meta *string) {
	t.Helper()
	var metadataValue any
	if meta != nil {
		metadataValue = *meta
	}
	_, err := pool.Exec(context.Background(),
		"INSERT INTO events (id, stream_id, version, type, schema_version, data, metadata) VALUES ($1, $2, $3, $4, 1, $5, $6)",
		uuid.NewString(), streamID, version, typeName, data, metadataValue,
	)
	require.NoError(t, err)
}
