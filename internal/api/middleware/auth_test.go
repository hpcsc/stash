//go:build unit

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/stash/internal/api/middleware"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	issuer := session.New([]byte("test-secret"), time.Hour)

	newRouter := func() chi.Router {
		router := chi.NewRouter()

		router.Group(func(r chi.Router) {
			r.Use(middleware.NewAuthMiddleware(issuer))
			r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("success"))
			})
		})

		return router
	}

	t.Run("returns Unauthorized when the token is missing", func(t *testing.T) {
		w := httptest.NewRecorder()

		newRouter().ServeHTTP(w, httptest.NewRequest("GET", "/protected", nil))

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns Unauthorized when the token is invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		newRouter().ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns Ok and sets the subject when the token is valid", func(t *testing.T) {
		w := httptest.NewRecorder()
		token, err := issuer.Issue("google-oauth2-123")
		require.NoError(t, err)
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		newRouter().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "success", w.Body.String())
	})
}
