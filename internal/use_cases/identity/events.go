package identity

import (
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/identity/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.SignedIn]()
	reg.Register[event.SignedOut]()
}
