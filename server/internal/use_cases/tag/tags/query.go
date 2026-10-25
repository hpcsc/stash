package tags

import (
	"context"
	"errors"
	"fmt"

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

func (q *Query) List(ctx context.Context) ([]Tag, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id::text, name FROM tags ORDER BY LOWER(name)
	`)
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

func (q *Query) ByID(ctx context.Context, id string) (*Tag, error) {
	var t Tag
	err := q.pool.QueryRow(ctx, `
		SELECT id::text, name FROM tags WHERE id = $1
	`, id).Scan(&t.ID, &t.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read tag %s: %w", id, err)
	}
	return &t, nil
}