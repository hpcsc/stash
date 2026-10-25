package identity

import (
	"errors"
	"fmt"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/identity/event"
)

var ErrNotSignedIn = errors.New("not signed in")

var _ domain.Aggregate = (*Account)(nil)

func StreamID(subject string) string {
	return fmt.Sprintf("identity-%s", subject)
}

type Account struct {
	domain.AggregateBase

	subject  string
	provider string
	signedIn bool
}

func (a *Account) ID() string {
	return a.subject
}

func (a *Account) SignIn(subject, provider string) error {
	if subject == "" {
		return errors.New("subject must not be empty")
	}
	if provider == "" {
		return errors.New("provider must not be empty")
	}
	if a.signedIn {
		// Already signed in: a re-authentication, which only refreshes the
		// session token. It records no new SignedIn event.
		return nil
	}

	a.record(domain.NewEvent(&event.SignedIn{
		Subject:  subject,
		Provider: provider,
	}))
	return nil
}

func (a *Account) SignOut() error {
	if !a.signedIn {
		return ErrNotSignedIn
	}

	a.record(domain.NewEvent(&event.SignedOut{
		Subject: a.subject,
	}))
	return nil
}

func (a *Account) Apply(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.SignedIn:
		a.subject = p.Subject
		a.provider = p.Provider
		a.signedIn = true
	case *event.SignedOut:
		a.signedIn = false
	}
}

func (a *Account) record(e domain.Event) {
	a.Apply(e)
	a.Record(e)
}
