//go:build unit

package identity_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hpcsc/stash/internal/api/middleware"
	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/session"
	apiTest "github.com/hpcsc/stash/internal/api/test"
	"github.com/hpcsc/stash/internal/api/use_cases/identity"
	commonTest "github.com/hpcsc/stash/internal/common/test"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/command/fake"
	identityCommand "github.com/hpcsc/stash/internal/use_cases/identity/command"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	issuer := session.New([]byte("test-secret"), time.Hour)

	t.Run("sign in", func(t *testing.T) {
		t.Run("dispatches a sign-in command and returns a session token", func(t *testing.T) {
			handler := fake.NewHandler[identityCommand.SignIn]()
			bus := command.NewBus()
			bus.Register[identityCommand.SignIn](handler)
			router := newRouter(bus, issuer, fakeVerifier{subject: subject})

			w := send(router, "POST", "/auth/signin", `{"provider": "google", "idToken": "google-token"}`)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, identityCommand.SignIn{Subject: subject, Provider: "google"}, handler.TriggeredWithCommand().Payload)
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			token := body.Data.(map[string]any)["token"].(string)
			got, err := issuer.Verify(token)
			require.NoError(t, err)
			require.Equal(t, subject, got)
		})

		t.Run("rejects an unsupported provider", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[identityCommand.SignIn](fake.NewHandler[identityCommand.SignIn]())
			router := newRouter(bus, issuer, fakeVerifier{subject: subject})

			w := send(router, "POST", "/auth/signin", `{"provider": "facebook", "idToken": "x"}`)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run("rejects an ID token the verifier does not accept", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[identityCommand.SignIn](fake.NewHandler[identityCommand.SignIn]())
			router := newRouter(bus, issuer, fakeVerifier{err: errors.New("invalid token")})

			w := send(router, "POST", "/auth/signin", `{"provider": "google", "idToken": "bad"}`)

			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("rejects a body without an ID token", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[identityCommand.SignIn](fake.NewHandler[identityCommand.SignIn]())
			router := newRouter(bus, issuer, fakeVerifier{subject: subject})

			w := send(router, "POST", "/auth/signin", `{"provider": "google"}`)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("sign out", func(t *testing.T) {
		t.Run("dispatches a sign-out command for the signed-in subject", func(t *testing.T) {
			handler := fake.NewHandler[identityCommand.SignOut]()
			bus := command.NewBus()
			bus.Register[identityCommand.SignOut](handler)
			router := newRouter(bus, issuer, fakeVerifier{subject: subject})

			w := sendWithSubject(router, "POST", "/auth/signout", subject, "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, identityCommand.SignOut{Subject: subject}, handler.TriggeredWithCommand().Payload)
		})
	})
}

const subject = "google-oauth2-123"

type fakeVerifier struct {
	subject string
	err     error
}

func (v fakeVerifier) Verify(context.Context, string) (string, error) {
	return v.subject, v.err
}

func newRouter(bus *command.Bus, issuer *session.Issuer, verifier identity.IDTokenVerifier) http.Handler {
	return apiTest.NewRouterWithRoutable(identity.NewHandler(bus, map[string]identity.IDTokenVerifier{"google": verifier}, issuer))
}

func send(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	return sendWithSubject(router, method, path, "", body)
}

func sendWithSubject(router http.Handler, method, path, subject, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(middleware.WithSubject(req.Context(), subject))
	router.ServeHTTP(w, req)
	return w
}
