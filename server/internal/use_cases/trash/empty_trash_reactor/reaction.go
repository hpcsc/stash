package empty_trash_reactor

import (
	"context"
	"errors"
	"log/slog"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/reaction"
	"github.com/hpcsc/stash/internal/domain/reaction/dispatch"
	"github.com/hpcsc/stash/internal/use_cases/item"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/trash/event"
)

type Reaction struct {
	bus    *command.Bus
	logger *slog.Logger
}

var _ reaction.Interface = (*Reaction)(nil)

func New(bus *command.Bus, logger *slog.Logger) *Reaction {
	return &Reaction{
		bus:    bus,
		logger: logger,
	}
}

func (r *Reaction) Name() string {
	return "empty-trash"
}

func (r *Reaction) EventTypes() []string {
	return []string{event.TrashEmptied{}.EventType()}
}

func (r *Reaction) Handle(ctx context.Context, evt *domain.Event) error {
	emptied, ok := evt.Payload.(*event.TrashEmptied)
	if !ok {
		return nil
	}

	for _, itemID := range emptied.ItemIDs {
		cmd := domain.Command[itemCommand.Delete]{
			ID:      dispatch.CommandIDForTarget(r.Name(), evt, itemID.String()),
			Payload: itemCommand.Delete{ItemID: itemID},
		}
		err := r.bus.Dispatch(ctx, cmd)
		if err == nil || errors.Is(err, item.ErrNotCaptured) || errors.Is(err, item.ErrNotDropped) {
			continue
		}
		r.logger.Warn("failed to delete item",
			"itemId", itemID,
			"error", err,
		)
	}
	return nil
}
