package command

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
)

type Handler[P any] interface {
	Handle(ctx context.Context, cmd domain.Command[P]) error
}
