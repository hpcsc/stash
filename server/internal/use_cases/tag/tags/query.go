package tags

import (
	"context"
	"errors"
	"fmt"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewQuery(pool *pgxpool.Pool) *Query {
	return &Query{pool: pool}
}

type Query struct {
	pool *pgxpool.Pool
}

func (q *Query) For(user domain.UserID) View {
	return &Reader{
		pool: q.pool,
		user: user,
	}
}

type View interface {
	List(ctx context.Context) ([]Tag, error)
	ByID(ctx context.Context, id string) (*Tag, error)
}

type Reader struct {
	pool *pgxpool.Pool
	user domain.UserID
}

func (r *Reader) List(ctx context.Context) ([]Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name FROM tags WHERE user_id = $1 ORDER BY LOWER(name)
	`, r.user.String())
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Tag, error) {
		var t Tag
		err := r.Scan(&t.ID, &t.Name)
		return t, err
	})
}

func (r *Reader) ByID(ctx context.Context, id string) (*Tag, error) {
	var t Tag
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name FROM tags WHERE id = $1 AND user_id = $2
	`, id, r.user.String()).Scan(&t.ID, &t.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read tag %s: %w", id, err)
	}
	return &t, nil
}
