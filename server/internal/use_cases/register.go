package use_cases

import (
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/identity"
	"github.com/hpcsc/stash/internal/use_cases/identity/sign_in"
	"github.com/hpcsc/stash/internal/use_cases/identity/sign_out"
	"github.com/hpcsc/stash/internal/use_cases/item"
	"github.com/hpcsc/stash/internal/use_cases/item/capture_file"
	"github.com/hpcsc/stash/internal/use_cases/item/capture_image"
	"github.com/hpcsc/stash/internal/use_cases/item/capture_link"
	"github.com/hpcsc/stash/internal/use_cases/item/capture_note"
	"github.com/hpcsc/stash/internal/use_cases/item/capture_text"
	"github.com/hpcsc/stash/internal/use_cases/item/delete_item"
	"github.com/hpcsc/stash/internal/use_cases/item/drop"
	"github.com/hpcsc/stash/internal/use_cases/item/keep"
	"github.com/hpcsc/stash/internal/use_cases/item/restore"
	"github.com/hpcsc/stash/internal/use_cases/tag"
	"github.com/hpcsc/stash/internal/use_cases/tag/create_tag"
	"github.com/hpcsc/stash/internal/use_cases/tag/rename_tag"
	"github.com/hpcsc/stash/internal/use_cases/trash"
	"github.com/hpcsc/stash/internal/use_cases/trash/empty_trash"
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

	sign_in.Register(bus, identities)
	sign_out.Register(bus, identities)

	capture_text.Register(bus, items)
	capture_link.Register(bus, items)
	capture_image.Register(bus, items)
	capture_file.Register(bus, items)
	capture_note.Register(bus, items)
	keep.Register(bus, items)
	drop.Register(bus, items)
	restore.Register(bus, items)
	delete_item.Register(bus, items)

	create_tag.Register(bus, tags)
	rename_tag.Register(bus, tags)

	empty_trash.Register(bus, trashRepo)
}
