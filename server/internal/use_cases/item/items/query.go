package items

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Body       string    `json:"body"`
	State      string    `json:"state"`
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

func (q *Query) List(ctx context.Context, search string, states ...string) ([]Item, error) {
	query := `
		SELECT id::text, type, body, state, source_app, source_url, captured_at
		FROM items
	`
	var args []any
	var clauses []string
	argIdx := 1

	if search != "" {
		clauses = append(clauses, fmt.Sprintf("body ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, search)
		argIdx++
	}

	if len(states) > 0 {
		placeholders := make([]string, len(states))
		for i, s := range states {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, s)
			argIdx++
		}
		clauses = append(clauses, fmt.Sprintf("state IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY captured_at DESC"

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) {
		var i Item
		err := r.Scan(&i.ID, &i.Type, &i.Body, &i.State, &i.SourceApp, &i.SourceURL, &i.CapturedAt)
		return i, err
	})
}

func (q *Query) ByID(ctx context.Context, id string) (*Item, error) {
	var i Item
	err := q.pool.QueryRow(ctx, `
		SELECT id::text, type, body, state, source_app, source_url, captured_at
		FROM items
		WHERE id = $1
	`, id).Scan(&i.ID, &i.Type, &i.Body, &i.State, &i.SourceApp, &i.SourceURL, &i.CapturedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read item %s: %w", id, err)
	}
	return &i, nil
}