//go:build unit

package events_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/session"
	apiTest "github.com/hpcsc/stash/internal/api/test"
	"github.com/hpcsc/stash/internal/api/use_cases/events"
	commonTest "github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/domain/fake"
	"github.com/stretchr/testify/require"
)

const adminToken = "0123456789abcdef0123456789abcdef"

var testUser = domain.UserID("test-user")

func TestHandler(t *testing.T) {
	t.Run("scope", func(t *testing.T) {
		t.Run("returns Unauthorized without a token", func(t *testing.T) {
			router := newRouter(t, &fakeSource{reader: &fakeReader{}})

			w := send(router, "GET", "/events", "")

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("returns Ok with all scope for a matching admin token in the X-Debug-Token header", func(t *testing.T) {
			source := &fakeSource{reader: &fakeReader{types: []string{"stash.something-happened"}}}
			router := newRouter(t, source)

			w := sendWithHeaders(router, "GET", "/events/types", map[string]string{"X-Debug-Token": adminToken})

			require.Equal(t, http.StatusOK, w.Code)
			require.Len(t, source.scopes, 1)
			require.True(t, source.scopes[0].IsAll())
		})

		t.Run("returns Ok with all scope for a matching admin token in the Authorization header", func(t *testing.T) {
			source := &fakeSource{reader: &fakeReader{types: []string{"stash.something-happened"}}}
			router := newRouter(t, source)

			w := sendWithHeaders(router, "GET", "/events/types", map[string]string{"Authorization": "Bearer " + adminToken})

			require.Equal(t, http.StatusOK, w.Code)
			require.True(t, source.scopes[0].IsAll())
		})

		t.Run("ignores an admin token in the query", func(t *testing.T) {
			source := &fakeSource{reader: &fakeReader{}}
			router := newRouter(t, source)

			w := send(router, "GET", "/events/types?debugToken="+adminToken, "")

			require.Equal(t, http.StatusUnauthorized, w.Code)
			require.Empty(t, source.scopes)
		})

		t.Run("returns Ok with the user scope for a valid session token", func(t *testing.T) {
			source := &fakeSource{reader: &fakeReader{types: []string{"stash.something-happened"}}}
			router := newRouter(t, source)

			w := sendWithHeaders(router, "GET", "/events/types", map[string]string{"Authorization": "Bearer " + sessionToken(t)})

			require.Equal(t, http.StatusOK, w.Code)
			require.Len(t, source.scopes, 1)
			require.False(t, source.scopes[0].IsAll())
			require.Equal(t, testUser, source.scopes[0].User())
		})

		t.Run("falls through to the session when the admin token does not match", func(t *testing.T) {
			source := &fakeSource{reader: &fakeReader{types: []string{"stash.something-happened"}}}
			router := newRouter(t, source)

			w := sendWithHeaders(router, "GET", "/events/types", map[string]string{
				"X-Debug-Token": "wrong-token",
				"Authorization": "Bearer " + sessionToken(t),
			})

			require.Equal(t, http.StatusOK, w.Code)
			require.False(t, source.scopes[0].IsAll())
			require.Equal(t, testUser, source.scopes[0].User())
		})
	})

	t.Run("list", func(t *testing.T) {
		t.Run("returns decoded events and the next cursor", func(t *testing.T) {
			stored := storedEvent(1, 1, "stash.something-happened", `{"ID":"a","Value":"hello"}`)
			source := &fakeSource{reader: &fakeReader{
				list: []events.StoredEvent{stored},
				next: &domain.Position{TransactionID: 1, Sequence: 1},
			}}
			router := newRouter(t, source)

			w := send(router, "GET", "/events", sessionToken(t))

			require.Equal(t, http.StatusOK, w.Code)
			page := dataOf[events.Page](t, w)
			require.Len(t, page.Events, 1)
			require.Equal(t, "stash.something-happened", page.Events[0].Type)
			require.Contains(t, string(page.Events[0].Decoded), `"Value":"hello"`)
			require.Empty(t, page.Events[0].ParseError)
			require.Equal(t, uint64(1), page.Next.TransactionID)
		})

		t.Run("surfaces a parse error without failing the request", func(t *testing.T) {
			stored := storedEvent(1, 1, "stash.unknown-type", `{"x":1}`)
			source := &fakeSource{reader: &fakeReader{list: []events.StoredEvent{stored}}}
			router := newRouter(t, source)

			w := send(router, "GET", "/events", sessionToken(t))

			require.Equal(t, http.StatusOK, w.Code)
			page := dataOf[events.Page](t, w)
			require.Contains(t, page.Events[0].ParseError, "stash.unknown-type")
			require.Equal(t, json.RawMessage(`{"x":1}`), page.Events[0].Data)
		})

		t.Run("passes the filters through to the reader", func(t *testing.T) {
			reader := &fakeReader{list: []events.StoredEvent{}}
			source := &fakeSource{reader: reader}
			router := newRouter(t, source)

			since := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
			until := since.Add(24 * time.Hour)
			w := send(router, "GET",
				"/events?afterTxid=7&afterSeq=9&limit=25&type=stash.a&type=stash.b&streamPrefix=user/alice&correlationId=abc&since="+since.Format(time.RFC3339)+"&until="+until.Format(time.RFC3339),
				sessionToken(t),
			)

			require.Equal(t, http.StatusOK, w.Code)
			filter := reader.filter
			require.Equal(t, &domain.Position{TransactionID: 7, Sequence: 9}, filter.After)
			require.Equal(t, 25, filter.Limit)
			require.Equal(t, []string{"stash.a", "stash.b"}, filter.Types)
			require.Equal(t, "user/alice", filter.StreamPrefix)
			require.Equal(t, "abc", filter.CorrelationID)
			require.Equal(t, since, *filter.Since)
			require.Equal(t, until, *filter.Until)
		})

		t.Run("rejects an invalid limit", func(t *testing.T) {
			router := newRouter(t, &fakeSource{reader: &fakeReader{}})

			w := send(router, "GET", "/events?limit=nope", sessionToken(t))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("stream", func(t *testing.T) {
		t.Run("returns a stream's events", func(t *testing.T) {
			reader := &fakeReader{stream: []events.StoredEvent{storedEvent(1, 1, "stash.something-happened", `{"ID":"a","Value":"x"}`)}}
			source := &fakeSource{reader: reader}
			router := newRouter(t, source)

			w := send(router, "GET", "/events/stream?streamId=user/alice/item-1", sessionToken(t))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "user/alice/item-1", reader.streamID)
			page := dataOf[[]events.Event](t, w)
			require.Len(t, page, 1)
		})

		t.Run("returns not found when the stream has no events", func(t *testing.T) {
			router := newRouter(t, &fakeSource{reader: &fakeReader{stream: []events.StoredEvent{}}})

			w := send(router, "GET", "/events/stream?streamId=user/alice/item-1", sessionToken(t))

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})

	t.Run("types", func(t *testing.T) {
		t.Run("returns the distinct types", func(t *testing.T) {
			router := newRouter(t, &fakeSource{reader: &fakeReader{types: []string{"stash.a", "stash.b"}}})

			w := send(router, "GET", "/events/types", sessionToken(t))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, []string{"stash.a", "stash.b"}, dataOf[[]string](t, w))
		})
	})

	t.Run("streams", func(t *testing.T) {
		t.Run("returns the streams", func(t *testing.T) {
			router := newRouter(t, &fakeSource{reader: &fakeReader{streams: []events.StreamSummary{
				{StreamID: "user/a/item-1", EventCount: 3, LastVersion: 3},
			}}})

			w := send(router, "GET", "/events/streams", sessionToken(t))

			require.Equal(t, http.StatusOK, w.Code)
			streams := dataOf[[]events.StreamSummary](t, w)
			require.Len(t, streams, 1)
			require.Equal(t, "user/a/item-1", streams[0].StreamID)
			require.Equal(t, int64(3), streams[0].EventCount)
		})
	})

	t.Run("checkpoints", func(t *testing.T) {
		t.Run("computes the lag from the last position", func(t *testing.T) {
			reader := &fakeReader{
				checkpoints: []events.CheckpointRow{{Name: "items", Position: domain.Position{TransactionID: 1, Sequence: 4}}},
				head:        &domain.Position{TransactionID: 2, Sequence: 10},
			}
			router := newRouter(t, &fakeSource{reader: reader})

			w := send(router, "GET", "/events/checkpoints", sessionToken(t))

			require.Equal(t, http.StatusOK, w.Code)
			view := dataOf[events.Checkpoints](t, w)
			require.Equal(t, uint64(2), view.LastPosition.TransactionID)
			require.Len(t, view.Checkpoints, 1)
			require.Equal(t, uint64(6), view.Checkpoints[0].Lag)
		})
	})

	t.Run("tail", func(t *testing.T) {
		t.Run("writes events as server-sent events", func(t *testing.T) {
			reader := &fakeReader{tail: []events.StoredEvent{storedEvent(1, 1, "stash.something-happened", `{"ID":"a","Value":"live"}`)}}
			router := newRouter(t, &fakeSource{reader: reader})

			w := send(router, "GET", "/events/tail", sessionToken(t))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
			require.Contains(t, w.Body.String(), "data: ")
			require.Contains(t, w.Body.String(), `"Value":"live"`)
		})

		t.Run("returns Unauthorized without a token", func(t *testing.T) {
			router := newRouter(t, &fakeSource{reader: &fakeReader{}})

			w := send(router, "GET", "/events/tail", "")

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})
}

type fakeSource struct {
	reader *fakeReader
	scopes []events.Scope
}

func (f *fakeSource) Scoped(scope events.Scope) events.Reader {
	f.scopes = append(f.scopes, scope)
	return f.reader
}

type fakeReader struct {
	list        []events.StoredEvent
	next        *domain.Position
	listErr     error
	filter      events.ListFilter
	stream      []events.StoredEvent
	streamID    string
	streams     []events.StreamSummary
	types       []string
	checkpoints []events.CheckpointRow
	head        *domain.Position
	tail        []events.StoredEvent
}

func (f *fakeReader) List(_ context.Context, filter events.ListFilter) ([]events.StoredEvent, *domain.Position, error) {
	f.filter = filter
	if f.listErr != nil {
		return nil, nil, f.listErr
	}
	return f.list, f.next, nil
}

func (f *fakeReader) Stream(_ context.Context, streamID string) ([]events.StoredEvent, error) {
	f.streamID = streamID
	return f.stream, nil
}

func (f *fakeReader) Streams(context.Context) ([]events.StreamSummary, error) {
	return f.streams, nil
}

func (f *fakeReader) Types(context.Context) ([]string, error) {
	return f.types, nil
}

func (f *fakeReader) Checkpoints(context.Context) ([]events.CheckpointRow, *domain.Position, error) {
	return f.checkpoints, f.head, nil
}

func (f *fakeReader) Tail(_ context.Context, _ *domain.Position, emit func(events.StoredEvent) error) error {
	for _, event := range f.tail {
		if err := emit(event); err != nil {
			return err
		}
	}
	return nil
}

func newRouter(t *testing.T, source events.ReaderSource) http.Handler {
	t.Helper()
	issuer := session.New([]byte("test-secret"), time.Hour)
	reg := registry.New()
	reg.Register[fake.SomethingHappened]()
	return apiTest.NewRouterWithRoutable(events.NewHandler(source, reg, issuer, adminToken, commonTest.DiscardLogger()))
}

func sessionToken(t *testing.T) string {
	t.Helper()
	issuer := session.New([]byte("test-secret"), time.Hour)
	token, err := issuer.Issue(string(testUser))
	require.NoError(t, err)
	return token
}

func storedEvent(txid, sequence uint64, eventType, data string) events.StoredEvent {
	return events.StoredEvent{
		ID:            "event-id",
		StreamID:      "user/test-user/item-1",
		Version:       1,
		Position:      domain.Position{TransactionID: txid, Sequence: sequence},
		Type:          eventType,
		SchemaVersion: 1,
		Data:          []byte(data),
		CreatedAt:     time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
	}
}

func send(router http.Handler, method, path, token string) *httptest.ResponseRecorder {
	headers := map[string]string{}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	return sendWithHeaders(router, method, path, headers)
}

func sendWithHeaders(router http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	router.ServeHTTP(w, req)
	return w
}

func dataOf[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
	raw, err := json.Marshal(body.Data)
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}
