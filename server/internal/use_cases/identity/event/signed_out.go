package event

import domainEvent "github.com/hpcsc/stash/internal/domain/event"

type SignedOut struct {
	Subject string `json:"subject"`
}

func (SignedOut) EventType() string {
	return domainEvent.Namespace + ".signed-out"
}