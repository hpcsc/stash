//go:build unit

package event_browser_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/event_browser"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("proxy", func(t *testing.T) {
		t.Run("sends the token to the backend instead of the browser", func(t *testing.T) {
			var received string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"successful":true,"messages":[],"data":[]}`))
			}))
			defer backend.Close()

			handler := newHandler(t, backend.URL)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/events", nil)
			req.Header.Set("Authorization", "Bearer from-the-browser")

			handler.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "Bearer secret-token", received)
		})

		t.Run("forwards the query string and the path to the backend", func(t *testing.T) {
			var receivedURL string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedURL = r.URL.String()
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			handler := newHandler(t, backend.URL)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/events/stream?streamId=user%2Fa%2Fitem-1", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "/api/events/stream?streamId=user%2Fa%2Fitem-1", receivedURL)
		})

		t.Run("keeps the path prefix of the backend URL", func(t *testing.T) {
			var receivedPath string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			handler := newHandler(t, backend.URL+"/stash")

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/events/types", nil))

			require.Equal(t, "/stash/api/events/types", receivedPath)
		})

		t.Run("returns bad gateway when the backend is unreachable", func(t *testing.T) {
			handler := newHandler(t, "http://127.0.0.1:1")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/events", nil))

			require.Equal(t, http.StatusBadGateway, w.Code)
			require.Contains(t, w.Body.String(), "successful")
		})

		t.Run("streams the backend response without buffering it", func(t *testing.T) {
			release := make(chan struct{})
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, "data: first\n\n")
				w.(http.Flusher).Flush()
				<-release
			}))
			defer backend.Close()
			defer close(release)

			handler := newHandler(t, backend.URL)
			server := httptest.NewServer(handler)
			defer server.Close()

			resp, err := http.Get(server.URL + "/api/events/tail")
			require.NoError(t, err)
			defer resp.Body.Close()

			buffer := make([]byte, len("data: first\n\n"))
			_, err = io.ReadFull(resp.Body, buffer)
			require.NoError(t, err)
			require.Equal(t, "data: first\n\n", string(buffer))
		})
	})

	t.Run("ui", func(t *testing.T) {
		t.Run("serves the index page at the root", func(t *testing.T) {
			handler := newHandler(t, "http://127.0.0.1:1")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), "<div id=\"app\">")
		})

		t.Run("serves a built asset", func(t *testing.T) {
			handler := newHandler(t, "http://127.0.0.1:1")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/assets/index.js", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "console.log('event browser')", w.Body.String())
		})

		t.Run("falls back to the index page for an unknown path", func(t *testing.T) {
			handler := newHandler(t, "http://127.0.0.1:1")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/some/deep/link", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), "<div id=\"app\">")
		})

		t.Run("rejects a write request", func(t *testing.T) {
			handler := newHandler(t, "http://127.0.0.1:1")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("POST", "/", strings.NewReader("{}")))

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})
}

func newHandler(t *testing.T, backend string) http.Handler {
	t.Helper()
	handler, err := event_browser.NewHandler(backend, "secret-token", ui(), test.DiscardLogger())
	require.NoError(t, err)
	return handler
}

func ui() fstest.MapFS {
	return fstest.MapFS{
		"index.html":      {Data: []byte(`<!doctype html><html><body><div id="app"></div></body></html>`)},
		"assets/index.js": {Data: []byte("console.log('event browser')")},
	}
}
