package use_cases

import (
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/identity"
	"github.com/hpcsc/stash/internal/use_cases/identity/signin"
	"github.com/hpcsc/stash/internal/use_cases/identity/signout"
	"github.com/hpcsc/stash/internal/use_cases/item"
	"github.com/hpcsc/stash/internal/use_cases/item/capturetext"
	"github.com/hpcsc/stash/internal/use_cases/item/drop"
	"github.com/hpcsc/stash/internal/use_cases/item/keep"
)

func RegisterEvents(reg *registry.OfEvents) {
	identity.RegisterEvents(reg)
	item.RegisterEvents(reg)
}

func RegisterHandlers(bus *command.Bus, stream event.Stream) {
	identities := identity.NewRepository(stream)
	items := item.NewRepository(stream)

	signin.Register(bus, identities)
	signout.Register(bus, identities)

	capturetext.Register(bus, items)
	keep.Register(bus, items)
	drop.Register(bus, items)
}
