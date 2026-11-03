//go:build unit

package event_browser_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hpcsc/stash/internal/api/response"
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

		t.Run("uses the token that sign-in stored", func(t *testing.T) {
			var received string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			session := &fakeSession{token: "from-sign-in"}
			handler := newHandlerWithSession(t, backend.URL, session)

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/events", nil))

			require.Equal(t, "Bearer from-sign-in", received)
		})

		t.Run("clears the session when the backend rejects it", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer backend.Close()

			session := &fakeSession{token: "expired"}
			handler := newHandlerWithSession(t, backend.URL, session)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/events", nil))

			require.Equal(t, http.StatusUnauthorized, w.Code)
			require.True(t, session.cleared)
			require.Empty(t, session.token)
		})

		t.Run("keeps a token from the command line when the backend rejects it", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer backend.Close()

			session := &fakeSession{token: "from-command-line", fixed: true}
			handler := newHandlerWithSession(t, backend.URL, session)

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/events", nil))

			require.False(t, session.cleared)
			require.Equal(t, "from-command-line", session.token)
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

	t.Run("session", func(t *testing.T) {
		t.Run("returns the signed-out state and the client id", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/auth/config", r.URL.Path)
				writeEnvelope(t, w, response.SucceedWithData(map[string]string{"googleClientId": "the-client-id"}))
			}))
			defer backend.Close()

			handler := newHandlerWithSession(t, backend.URL, &fakeSession{})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("GET", "/session", nil))

			require.Equal(t, http.StatusOK, w.Code)
			data := readData(t, w)
			require.Equal(t, false, data["signedIn"])
			require.Equal(t, "the-client-id", data["googleClientId"])
			require.Equal(t, true, data["canSignOut"])
		})

		t.Run("returns the signed-in state", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeEnvelope(t, w, response.SucceedWithData(map[string]string{"googleClientId": "the-client-id"}))
			}))
			defer backend.Close()

			handler := newHandlerWithSession(t, backend.URL, &fakeSession{token: "stored"})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("GET", "/session", nil))

			require.Equal(t, true, readData(t, w)["signedIn"])
		})

		t.Run("hides sign-out when the token comes from the command line", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeEnvelope(t, w, response.SucceedWithData(map[string]string{"googleClientId": "the-client-id"}))
			}))
			defer backend.Close()

			handler := newHandlerWithSession(t, backend.URL, &fakeSession{token: "fixed", fixed: true})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("GET", "/session", nil))

			require.Equal(t, false, readData(t, w)["canSignOut"])
		})

		t.Run("exchanges an ID token and stores the session token", func(t *testing.T) {
			var receivedBody map[string]string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/auth/config":
					writeEnvelope(t, w, response.SucceedWithData(map[string]string{"googleClientId": "the-client-id"}))
				case r.Method == http.MethodPost && r.URL.Path == "/api/auth/signin":
					require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedBody))
					writeEnvelope(t, w, response.SucceedWithData(map[string]string{"token": "new-session"}))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer backend.Close()

			session := &fakeSession{}
			handler := newHandlerWithSession(t, backend.URL, session)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("POST", "/session", strings.NewReader(`{"idToken":"google-id-token"}`)))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, map[string]string{"provider": "google", "idToken": "google-id-token"}, receivedBody)
			require.Equal(t, "new-session", session.token)
			data := readData(t, w)
			require.Equal(t, true, data["signedIn"])
			require.Equal(t, "the-client-id", data["googleClientId"])
		})

		t.Run("relays a rejected ID token", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeEnvelopeStatus(t, w, http.StatusUnauthorized, response.Fail("sign-in failed"))
			}))
			defer backend.Close()

			session := &fakeSession{}
			handler := newHandlerWithSession(t, backend.URL, session)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("POST", "/session", strings.NewReader(`{"idToken":"bad"}`)))

			require.Equal(t, http.StatusUnauthorized, w.Code)
			require.Empty(t, session.token)
		})

		t.Run("rejects a request without an ID token", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer backend.Close()

			handler := newHandlerWithSession(t, backend.URL, &fakeSession{})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("POST", "/session", strings.NewReader(`{}`)))

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("signs out of the backend before clearing the session", func(t *testing.T) {
			var receivedPath, receivedAuth string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				receivedAuth = r.Header.Get("Authorization")
				writeEnvelope(t, w, response.Succeed())
			}))
			defer backend.Close()

			session := &fakeSession{token: "stored"}
			handler := newHandlerWithSession(t, backend.URL, session)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("DELETE", "/session", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "/api/auth/signout", receivedPath)
			require.Equal(t, "Bearer stored", receivedAuth)
			require.True(t, session.cleared)
			require.Empty(t, session.token)
		})

		t.Run("clears the session when the backend sign-out fails", func(t *testing.T) {
			session := &fakeSession{token: "stored"}
			handler := newHandlerWithSession(t, "http://127.0.0.1:1", session)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("DELETE", "/session", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.True(t, session.cleared)
		})

		t.Run("does not sign out the backend for a token from the command line", func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("the CLI must not call the backend for a fixed token")
			}))
			defer backend.Close()

			session := &fakeSession{token: "from-command-line", fixed: true}
			handler := newHandlerWithSession(t, backend.URL, session)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("DELETE", "/session", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.False(t, session.cleared)
			require.Equal(t, "from-command-line", session.token)
		})

		t.Run("returns bad gateway when the backend does not serve the client id", func(t *testing.T) {
			handler := newHandlerWithSession(t, "http://127.0.0.1:1", &fakeSession{})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, httptest.NewRequest("GET", "/session", nil))

			require.Equal(t, http.StatusBadGateway, w.Code)
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

type fakeSession struct {
	token   string
	fixed   bool
	cleared bool
}

func (s *fakeSession) Token() string {
	return s.token
}

func (s *fakeSession) Set(token string) error {
	s.token = token
	return nil
}

func (s *fakeSession) Clear() error {
	if s.fixed {
		return nil
	}
	s.token = ""
	s.cleared = true
	return nil
}

func (s *fakeSession) Fixed() bool {
	return s.fixed
}

func newHandler(t *testing.T, backend string) http.Handler {
	t.Helper()
	return newHandlerWithSession(t, backend, &fakeSession{token: "secret-token"})
}

func newHandlerWithSession(t *testing.T, backend string, session event_browser.Session) http.Handler {
	t.Helper()
	handler, err := event_browser.NewHandler(backend, session, ui(), test.DiscardLogger())
	require.NoError(t, err)
	return handler
}

func ui() fstest.MapFS {
	return fstest.MapFS{
		"index.html":      {Data: []byte(`<!doctype html><html><body><div id="app"></div></body></html>`)},
		"assets/index.js": {Data: []byte("console.log('event browser')")},
	}
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, body *response.Response) {
	t.Helper()
	writeEnvelopeStatus(t, w, http.StatusOK, body)
}

func writeEnvelopeStatus(t *testing.T, w http.ResponseWriter, status int, body *response.Response) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

func readData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	body := test.UnmarshalJson[response.Response](t, w.Body.Bytes())
	require.True(t, body.Successful)
	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	return data
}
