package use_cases

import (
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/identity"
	"github.com/hpcsc/stash/internal/use_cases/identity/signin"
	"github.com/hpcsc/stash/internal/use_cases/identity/signout"
	"github.com/hpcsc/stash/internal/use_cases/item"
	"github.com/hpcsc/stash/internal/use_cases/item/capturefile"
	"github.com/hpcsc/stash/internal/use_cases/item/captureimage"
	"github.com/hpcsc/stash/internal/use_cases/item/capturelink"
	"github.com/hpcsc/stash/internal/use_cases/item/capturenote"
	"github.com/hpcsc/stash/internal/use_cases/item/capturetext"
	"github.com/hpcsc/stash/internal/use_cases/item/deleteitem"
	"github.com/hpcsc/stash/internal/use_cases/item/drop"
	"github.com/hpcsc/stash/internal/use_cases/item/keep"
	"github.com/hpcsc/stash/internal/use_cases/item/restore"
	"github.com/hpcsc/stash/internal/use_cases/tag"
	"github.com/hpcsc/stash/internal/use_cases/tag/createtag"
	"github.com/hpcsc/stash/internal/use_cases/tag/renametag"
	"github.com/hpcsc/stash/internal/use_cases/trash"
	"github.com/hpcsc/stash/internal/use_cases/trash/emptytrash"
)

func RegisterEvents(reg *registry.OfEvents) {
	identity.RegisterEvents(reg)
	item.RegisterEvents(reg)
	tag.RegisterEvents(reg)
	trash.RegisterEvents(reg)
}

func RegisterHandlers(bus *command.Bus, stream event.Stream) {
	identities := identity.NewRepository(stream)
	items := item.NewRepository(stream)
	tags := tag.NewRepository(stream)
	trashRepo := trash.NewRepository(stream)

	signin.Register(bus, identities)
	signout.Register(bus, identities)

	capturetext.Register(bus, items)
	capturelink.Register(bus, items)
	captureimage.Register(bus, items)
	capturefile.Register(bus, items)
	capturenote.Register(bus, items)
	keep.Register(bus, items)
	drop.Register(bus, items)
	restore.Register(bus, items)
	deleteitem.Register(bus, items)

	createtag.Register(bus, tags)
	renametag.Register(bus, tags)

	emptytrash.Register(bus, trashRepo)
}
