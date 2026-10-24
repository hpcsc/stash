package items

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Body       string    `json:"body"`
	SourceApp  string    `json:"sourceApp,omitempty"`
	SourceURL  string    `json:"sourceUrl,omitempty"`
	CapturedAt time.Time `json:"capturedAt"`
}

func NewQuery(pool *pgxpool.Pool) *Query {
	return &Query{pool: pool}
}

type Query struct {
	pool *pgxpool.Pool
}

func (q *Query) List(ctx context.Context) ([]Item, error) {
	rows, err := q.pool.Query(ctx, `
		SELECT id::text, type, body, source_app, source_url, captured_at
		FROM items
		ORDER BY captured_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) {
		var i Item
		err := r.Scan(&i.ID, &i.Type, &i.Body, &i.SourceApp, &i.SourceURL, &i.CapturedAt)
		return i, err
	})
}

func (q *Query) ByID(ctx context.Context, id string) (*Item, error) {
	var i Item
	err := q.pool.QueryRow(ctx, `
		SELECT id::text, type, body, source_app, source_url, captured_at
		FROM items
		WHERE id = $1
	`, id).Scan(&i.ID, &i.Type, &i.Body, &i.SourceApp, &i.SourceURL, &i.CapturedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read item %s: %w", id, err)
	}
	return &i, nil
}
