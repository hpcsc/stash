package trash

import (
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/trash/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.TrashEmptied]()
}
