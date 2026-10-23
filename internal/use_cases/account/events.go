package account

import (
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/account/event"
)

func RegisterEvents(reg *registry.OfEvents) {
	reg.Register[event.AccountOpened]()
	reg.Register[event.MoneyDeposited]()
	reg.Register[event.MoneyWithdrawn]()
	reg.Register[event.WelcomeBonusGranted]()
	reg.Register[event.AccountClosed]()
}
