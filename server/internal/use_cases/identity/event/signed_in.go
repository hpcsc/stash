package event

import domainEvent "github.com/hpcsc/stash/internal/domain/event"

type SignedIn struct {
	Subject  string `json:"subject"`
	Provider string `json:"provider"`
}

func (SignedIn) EventType() string {
	return domainEvent.Namespace + ".signed-in"
}