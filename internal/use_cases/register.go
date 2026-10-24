package use_cases

import (
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/identity"
	"github.com/hpcsc/stash/internal/use_cases/identity/signin"
	"github.com/hpcsc/stash/internal/use_cases/identity/signout"
)

func RegisterEvents(reg *registry.OfEvents) {
	identity.RegisterEvents(reg)
}

func RegisterHandlers(bus *command.Bus, stream event.Stream) {
	identities := identity.NewRepository(stream)

	signin.Register(bus, identities)
	signout.Register(bus, identities)
}
