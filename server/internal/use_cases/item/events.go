package item

import (
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.TextCaptured]()
	reg.Register[event.LinkCaptured]()
	reg.Register[event.LinkMetadataFetched]()
	reg.Register[event.ItemKept]()
	reg.Register[event.ItemDropped]()
	reg.Register[event.ItemTagged]()
}
