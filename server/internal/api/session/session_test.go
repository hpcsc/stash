//go:build unit

package session_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/stretchr/testify/require"
)

func TestIssuer(t *testing.T) {
	secret := []byte("test-secret")
	ttl := time.Hour

	t.Run("issue and verify", func(t *testing.T) {
		i := session.New(secret, ttl)

		raw, err := i.Issue(subject)
		require.NoError(t, err)

		got, err := i.Verify(raw)
		require.NoError(t, err)
		require.Equal(t, subject, got)
	})

	t.Run("verify", func(t *testing.T) {
		t.Run("rejects a token signed with another secret", func(t *testing.T) {
			raw, err := session.New([]byte("other-secret"), ttl).Issue(subject)
			require.NoError(t, err)

			_, err = session.New(secret, ttl).Verify(raw)
			require.ErrorIs(t, err, session.ErrInvalidToken)
		})

		t.Run("rejects an expired token", func(t *testing.T) {
			_, err := session.New(secret, ttl).Verify(expired(t, secret))
			require.ErrorIs(t, err, session.ErrInvalidToken)
		})

		t.Run("rejects a token that is not a JWT", func(t *testing.T) {
			_, err := session.New(secret, ttl).Verify("not-a-token")
			require.ErrorIs(t, err, session.ErrInvalidToken)
		})
	})
}

const subject = "google-oauth2-123"

func expired(t *testing.T, secret []byte) string {
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	require.NoError(t, err)
	return raw
}
