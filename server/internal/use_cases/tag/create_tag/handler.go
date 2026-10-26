package create_tag

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/use_cases/tag"
	tagCommand "github.com/hpcsc/stash/internal/use_cases/tag/command"
)

func Register(bus *command.Bus, tags *tag.Repository) {
	bus.Register[tagCommand.CreateTag](handler{tags: tags})
}

type handler struct {
	tags *tag.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[tagCommand.CreateTag]) error {
	t, err := h.tags.Load(ctx, cmd.Payload.TagID)
	if err != nil {
		return err
	}
	if err := t.Create(cmd.Payload); err != nil {
		return err
	}
	return h.tags.Save(ctx, t, cmd.ID)
}
