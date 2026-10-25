//go:build unit

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/stash/internal/api/middleware"
	"github.com/stretchr/testify/require"
)

func TestCORSMiddleware(t *testing.T) {
	newRouter := func() chi.Router {
		router := chi.NewRouter()
		router.Use(middleware.NewCORSMiddleware([]string{"http://localhost:8080"}))
		router.Get("/items", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
		router.Options("/*", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		return router
	}

	t.Run("sets the CORS headers for an allowed origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/items", nil)
		req.Header.Set("Origin", "http://localhost:8080")

		newRouter().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "http://localhost:8080", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("does not set the CORS headers for a disallowed origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/items", nil)
		req.Header.Set("Origin", "http://evil.example")

		newRouter().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("answers a preflight request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", "/auth/signin", nil)
		req.Header.Set("Origin", "http://localhost:8080")
		req.Header.Set("Access-Control-Request-Method", "POST")

		newRouter().ServeHTTP(w, req)

		require.Equal(t, http.StatusNoContent, w.Code)
		require.Equal(t, "http://localhost:8080", w.Header().Get("Access-Control-Allow-Origin"))
		require.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		require.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	})
}
