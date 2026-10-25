package tag

import (
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/tag/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.TagCreated]()
	reg.Register[event.TagRenamed]()
}