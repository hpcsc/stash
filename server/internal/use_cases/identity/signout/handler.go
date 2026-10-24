package signout

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/use_cases/identity"
	identityCommand "github.com/hpcsc/stash/internal/use_cases/identity/command"
)

func Register(bus *command.Bus, accounts *identity.Repository) {
	bus.Register[identityCommand.SignOut](handler{accounts: accounts})
}

type handler struct {
	accounts *identity.Repository
}

func (h handler) Handle(ctx context.Context, cmd domain.Command[identityCommand.SignOut]) error {
	a, err := h.accounts.Load(ctx, cmd.Payload.Subject)
	if err != nil {
		return err
	}
	if err := a.SignOut(); err != nil {
		return err
	}
	return h.accounts.Save(ctx, a, cmd.ID)
}
