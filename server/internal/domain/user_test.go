//go:build unit

package domain_test

import (
	"testing"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestUser(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		t.Run("names a stream under the user", func(t *testing.T) {
			require.Equal(t, "user/123/item-abc", domain.UserStream("123", "item-abc"))
		})

		t.Run("reads the user back out of a stream id", func(t *testing.T) {
			user, err := domain.UserOf("user/123/item-abc")

			require.NoError(t, err)
			require.Equal(t, domain.UserID("123"), user)
		})

		t.Run("rejects a stream without a user prefix", func(t *testing.T) {
			_, err := domain.UserOf("item-abc")

			require.ErrorIs(t, err, domain.ErrNotUserStream)
		})

		t.Run("rejects a malformed user stream", func(t *testing.T) {
			_, err := domain.UserOf("user//item-abc")

			require.ErrorIs(t, err, domain.ErrNotUserStream)
		})
	})

	t.Run("valid", func(t *testing.T) {
		t.Run("accepts a non-empty id without a slash", func(t *testing.T) {
			require.True(t, domain.UserID("123").Valid())
		})

		t.Run("rejects an empty id", func(t *testing.T) {
			require.False(t, domain.UserID("").Valid())
		})

		t.Run("rejects an id containing a slash", func(t *testing.T) {
			require.False(t, domain.UserID("a/b").Valid())
		})
	})
}
