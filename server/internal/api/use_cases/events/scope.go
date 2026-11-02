package events

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/hpcsc/stash/internal/api/session"
	"github.com/hpcsc/stash/internal/domain"
)

type Scope struct {
	all  bool
	user domain.UserID
}

func (s Scope) IsAll() bool {
	return s.all
}

func (s Scope) User() domain.UserID {
	return s.user
}

func AllScope() Scope {
	return Scope{all: true}
}

func UserScope(user domain.UserID) Scope {
	return Scope{user: user}
}

func resolveScope(r *http.Request, issuer *session.Issuer, debugToken string, eventBrowserUsers map[string]struct{}) (Scope, bool) {
	if debugToken != "" {
		for _, candidate := range []string{
			r.Header.Get("X-Debug-Token"),
			bearer(r),
		} {
			if candidate != "" && equalSecret(candidate, debugToken) {
				return AllScope(), true
			}
		}
	}

	subject, err := issuer.Verify(sessionToken(r))
	user := domain.UserID(subject)
	if err != nil || !user.Valid() {
		return Scope{}, false
	}
	if _, ok := eventBrowserUsers[subject]; ok {
		return AllScope(), true
	}
	return UserScope(user), true
}

func userIDSet(ids []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			set[id] = struct{}{}
		}
	}
	return set
}

func equalSecret(given, expected string) bool {
	if len(given) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(expected)) == 1
}

func sessionToken(r *http.Request) string {
	if token := bearer(r); token != "" {
		return token
	}
	return r.URL.Query().Get("token")
}

func bearer(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}

type scopeContextKey struct{}

func WithScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

func ScopeFromContext(ctx context.Context) (Scope, bool) {
	scope, ok := ctx.Value(scopeContextKey{}).(Scope)
	return scope, ok
}
