package item

import (
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.LinkCaptured]()
	reg.Register[event.ImageCaptured]()
	reg.Register[event.FileCaptured]()
	reg.Register[event.NoteCaptured]()
	reg.Register[event.LinkMetadataFetched]()
	reg.Register[event.ThumbnailGenerated]()
	reg.Register[event.ItemKept]()
	reg.Register[event.ItemDropped]()
	reg.Register[event.ItemRestored]()
	reg.Register[event.ItemDeleted]()
	reg.Register[event.ItemTagged]()
}
