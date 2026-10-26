package capturefile

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/use_cases/item"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
)

func Register(bus *command.Bus, items *item.Repository) {
	bus.Register[itemCommand.CaptureFile](handler{items: items})
}

type handler struct {
	items *item.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[itemCommand.CaptureFile]) error {
	i, err := h.items.Load(ctx, cmd.Payload.ItemID)
	if err != nil {
		return err
	}
	if err := i.CaptureFile(cmd.Payload); err != nil {
		return err
	}
	return h.items.Save(ctx, i, cmd.ID)
}
