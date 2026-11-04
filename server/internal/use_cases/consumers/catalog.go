package consumers

import (
	"github.com/hpcsc/stash/internal/use_cases/item/item_tags"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	"github.com/hpcsc/stash/internal/use_cases/item/link_metadata"
	"github.com/hpcsc/stash/internal/use_cases/item/thumbnail"
	"github.com/hpcsc/stash/internal/use_cases/tag/tags"
	"github.com/hpcsc/stash/internal/use_cases/trash/empty_trash_reactor"
)

type Kind string

const (
	KindProjection Kind = "projection"
	KindReaction   Kind = "reaction"
)

type Description struct {
	Name       string
	Kind       Kind
	EventTypes []string
}

func Descriptions() []Description {
	return []Description{
		{Name: items.Name, Kind: KindProjection, EventTypes: items.EventTypes},
		{Name: tags.Name, Kind: KindProjection, EventTypes: tags.EventTypes},
		{Name: item_tags.Name, Kind: KindProjection, EventTypes: item_tags.EventTypes},
		{Name: link_metadata.Name, Kind: KindReaction, EventTypes: link_metadata.EventTypes},
		{Name: thumbnail.Name, Kind: KindReaction, EventTypes: thumbnail.EventTypes},
		{Name: empty_trash_reactor.Name, Kind: KindReaction, EventTypes: empty_trash_reactor.EventTypes},
	}
}
