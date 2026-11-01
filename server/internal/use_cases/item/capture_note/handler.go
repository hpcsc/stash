package capture_note

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/use_cases/item"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
)

func Register(bus *command.Bus, items *item.Repository) {
	bus.Register[itemCommand.CaptureNote](handler{items: items})
}

type handler struct {
	items *item.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[itemCommand.CaptureNote]) error {
	items := h.items.For(cmd.Actor)
	i, err := items.Load(ctx, cmd.Payload.ItemID)
	if err != nil {
		return err
	}
	if err := i.CaptureNote(cmd.Payload); err != nil {
		return err
	}
	return items.Save(ctx, i, cmd.ID)
}
