package restore

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/use_cases/item"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
)

func Register(bus *command.Bus, items *item.Repository) {
	bus.Register[itemCommand.Restore](handler{items: items})
}

type handler struct {
	items *item.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[itemCommand.Restore]) error {
	items := h.items.For(cmd.Actor)
	i, err := items.Load(ctx, cmd.Payload.ItemID)
	if err != nil {
		return err
	}
	if err := i.Restore(cmd.Payload); err != nil {
		return err
	}
	return items.Save(ctx, i, cmd.ID)
}
