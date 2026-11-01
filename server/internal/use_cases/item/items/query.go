package items

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Body          string    `json:"body"`
	State         string    `json:"state"`
	Title         string    `json:"title,omitempty"`
	Description   string    `json:"description,omitempty"`
	FilePath      string    `json:"filePath,omitempty"`
	ThumbnailPath string    `json:"thumbnailPath,omitempty"`
	FileName      string    `json:"fileName,omitempty"`
	FileSize      int64     `json:"fileSize,omitempty"`
	MimeType      string    `json:"mimeType,omitempty"`
	CapturedAt    time.Time `json:"capturedAt"`
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
	List(ctx context.Context, search string, states ...string) ([]Item, error)
	ByID(ctx context.Context, id string) (*Item, error)
	Existing(ctx context.Context, itemType, body string) (*Item, error)
}

type Reader struct {
	pool *pgxpool.Pool
	user domain.UserID
}

const itemColumns = `id::text, type, body, state, title, description, file_path, thumbnail_path, file_name, file_size, mime_type, captured_at`

func (r *Reader) List(ctx context.Context, search string, states ...string) ([]Item, error) {
	query := `
		SELECT ` + itemColumns + `
		FROM items
	`
	args := []any{r.user.String()}
	clauses := []string{"user_id = $1"}
	argIdx := 2

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

	query += " WHERE " + strings.Join(clauses, " AND ")
	query += " ORDER BY captured_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanItem)
}

func (r *Reader) ByID(ctx context.Context, id string) (*Item, error) {
	var i Item
	err := r.pool.QueryRow(ctx, `
		SELECT `+itemColumns+`
		FROM items
		WHERE id = $1 AND user_id = $2
	`, id, r.user.String()).Scan(&i.ID, &i.Type, &i.Body, &i.State, &i.Title, &i.Description, &i.FilePath, &i.ThumbnailPath, &i.FileName, &i.FileSize, &i.MimeType, &i.CapturedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read item %s: %w", id, err)
	}
	return &i, nil
}

func (r *Reader) Existing(ctx context.Context, itemType, body string) (*Item, error) {
	var i Item
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, state
		FROM items
		WHERE user_id = $1 AND type = $2 AND body = $3
		ORDER BY captured_at DESC
		LIMIT 1
	`, r.user.String(), itemType, body).Scan(&i.ID, &i.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find existing %s %s: %w", itemType, body, err)
	}
	return &i, nil
}

func scanItem(r pgx.CollectableRow) (Item, error) {
	var i Item
	err := r.Scan(&i.ID, &i.Type, &i.Body, &i.State, &i.Title, &i.Description, &i.FilePath, &i.ThumbnailPath, &i.FileName, &i.FileSize, &i.MimeType, &i.CapturedAt)
	return i, err
}
