package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/hpcsc/stash/internal/domain"
)

type userContextKey struct{}

func NewAuthMiddleware(issuer *session.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			subject, err := issuer.Verify(token)
			user := domain.UserID(subject)
			if err != nil || !user.Valid() {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
		})
	}
}

func WithUser(ctx context.Context, user domain.UserID) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) domain.UserID {
	user, _ := ctx.Value(userContextKey{}).(domain.UserID)
	return user
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(response.Fail("sign in to continue"))
}
