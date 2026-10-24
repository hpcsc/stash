package checkpoint

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	Get(ctx context.Context, name string) (*domain.Position, error)
	Set(ctx context.Context, name string, position domain.Position) error
	SetInTx(ctx context.Context, tx pgx.Tx, name string, position domain.Position) error
}
