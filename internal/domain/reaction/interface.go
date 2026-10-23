package reaction

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
)

type Interface interface {
	Handle(ctx context.Context, event *domain.Event) error
	EventTypes() []string
	Name() string
}
