package empty_trash

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/use_cases/trash"
	trashCommand "github.com/hpcsc/stash/internal/use_cases/trash/command"
)

func Register(bus *command.Bus, trashRepo *trash.Repository) {
	bus.Register[trashCommand.EmptyTrash](handler{trash: trashRepo})
}

type handler struct {
	trash *trash.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[trashCommand.EmptyTrash]) error {
	t, err := h.trash.Load(ctx)
	if err != nil {
		return err
	}
	if err := t.Empty(cmd.Payload); err != nil {
		return err
	}
	return h.trash.Save(ctx, t, cmd.ID)
}
