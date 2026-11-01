package rename_tag

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/use_cases/tag"
	tagCommand "github.com/hpcsc/stash/internal/use_cases/tag/command"
)

func Register(bus *command.Bus, tags *tag.Repository) {
	bus.Register[tagCommand.RenameTag](handler{tags: tags})
}

type handler struct {
	tags *tag.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[tagCommand.RenameTag]) error {
	tags := h.tags.For(cmd.Actor)
	t, err := tags.Load(ctx, cmd.Payload.TagID)
	if err != nil {
		return err
	}
	if err := t.Rename(cmd.Payload); err != nil {
		return err
	}
	return tags.Save(ctx, t, cmd.ID)
}
