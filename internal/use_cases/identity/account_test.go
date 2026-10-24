//go:build unit

package identity_test

import (
	"testing"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/identity"
	"github.com/hpcsc/stash/internal/use_cases/identity/event"
	"github.com/stretchr/testify/require"
)

func TestAccount(t *testing.T) {
	t.Run("sign in", func(t *testing.T) {
		t.Run("signs in with a provider", func(t *testing.T) {
			var a identity.Account

			require.NoError(t, a.SignIn(subject, "google"))

			require.Equal(t, &event.SignedIn{Subject: subject, Provider: "google"}, onlyRecorded(t, &a).Payload)
			require.Equal(t, subject, a.ID())
		})

		t.Run("rejects a second sign-in while signed in", func(t *testing.T) {
			a := given(signedIn())

			require.ErrorIs(t, a.SignIn(subject, "google"), identity.ErrAlreadySignedIn)
			require.Empty(t, a.Recorded())
		})

		t.Run("signs in again after signing out", func(t *testing.T) {
			a := given(signedIn(), signedOut())

			require.NoError(t, a.SignIn(subject, "google"))

			require.Equal(t, &event.SignedIn{Subject: subject, Provider: "google"}, onlyRecorded(t, a).Payload)
		})

		t.Run("rejects an empty subject", func(t *testing.T) {
			var a identity.Account

			require.Error(t, a.SignIn("", "google"))
			require.Empty(t, a.Recorded())
		})

		t.Run("rejects an empty provider", func(t *testing.T) {
			var a identity.Account

			require.Error(t, a.SignIn(subject, ""))
			require.Empty(t, a.Recorded())
		})
	})

	t.Run("sign out", func(t *testing.T) {
		t.Run("signs out a signed-in account", func(t *testing.T) {
			a := given(signedIn())

			require.NoError(t, a.SignOut())

			require.Equal(t, &event.SignedOut{Subject: subject}, onlyRecorded(t, a).Payload)
		})

		t.Run("rejects an account that is not signed in", func(t *testing.T) {
			var a identity.Account

			require.ErrorIs(t, a.SignOut(), identity.ErrNotSignedIn)
			require.Empty(t, a.Recorded())
		})
	})
}

const subject = "google-oauth2-1234567890"

func given(history ...domain.Event) *identity.Account {
	var a identity.Account
	for _, e := range history {
		a.Apply(e)
	}
	a.MarkCommitted(uint64(len(history)))
	return &a
}

func signedIn() domain.Event {
	return domain.NewEvent(&event.SignedIn{Subject: subject, Provider: "google"})
}

func signedOut() domain.Event {
	return domain.NewEvent(&event.SignedOut{Subject: subject})
}

func onlyRecorded(t *testing.T, a *identity.Account) domain.Event {
	recorded := a.Recorded()
	require.Len(t, recorded, 1)
	return recorded[0]
}
