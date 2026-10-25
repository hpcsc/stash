package item

import (
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.TextCaptured]()
	reg.Register[event.ItemKept]()
	reg.Register[event.ItemDropped]()
}
