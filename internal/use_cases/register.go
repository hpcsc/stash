package use_cases

import (
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/account"
	"github.com/hpcsc/stash/internal/use_cases/account/closeaccount"
	"github.com/hpcsc/stash/internal/use_cases/account/deposit"
	"github.com/hpcsc/stash/internal/use_cases/account/open"
	"github.com/hpcsc/stash/internal/use_cases/account/welcomebonus"
	"github.com/hpcsc/stash/internal/use_cases/account/withdraw"
)

func RegisterEvents(reg *registry.OfEvents) {
	account.RegisterEvents(reg)
}

func RegisterHandlers(bus *command.Bus, stream event.Stream) {
	accounts := account.NewRepository(stream)

	open.Register(bus, accounts)
	deposit.Register(bus, accounts)
	withdraw.Register(bus, accounts)
	welcomebonus.Register(bus, accounts)
	closeaccount.Register(bus, accounts)
}
