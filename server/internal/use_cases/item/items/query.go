package items

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DefaultLimit is the page size used when a request asks for no specific limit.
	DefaultLimit = 50
	// MaxLimit caps how many items a single page can return.
	MaxLimit = 200
)

type Item struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Body          string    `json:"body"`
	State         string    `json:"state"`
	Version       uint64    `json:"version"`
	Title         string    `json:"title,omitempty"`
	Description   string    `json:"description,omitempty"`
	FilePath      string    `json:"filePath,omitempty"`
	ThumbnailPath string    `json:"thumbnailPath,omitempty"`
	FileName      string    `json:"fileName,omitempty"`
	FileSize      int64     `json:"fileSize,omitempty"`
	MimeType      string    `json:"mimeType,omitempty"`
	CapturedAt    time.Time `json:"capturedAt"`
	TagNames      []string  `json:"tagNames,omitempty"`
}

type Filter struct {
	Search string
	Tags   []string
	States []string
	// Limit of zero or less returns every matching item.
	Limit  int
	Offset int
}

type Page struct {
	Items  []Item `json:"items"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
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
	List(ctx context.Context, filter Filter) (Page, error)
	ByID(ctx context.Context, id string) (*Item, error)
	Existing(ctx context.Context, itemType, body string) (*Item, error)
}

type Reader struct {
	pool *pgxpool.Pool
	user domain.UserID
}

const itemColumns = `items.id::text, items.type, items.body, items.state, items.version, items.title, items.description, items.file_path, items.thumbnail_path, items.file_name, items.file_size, items.mime_type, items.captured_at`

const tagNamesSubquery = `COALESCE(
	(SELECT json_agg(t.name ORDER BY LOWER(t.name))
	 FROM item_tags it JOIN tags t ON t.id = it.tag_id
	 WHERE it.item_id = items.id),
	'[]'::json
) AS tag_names`

func (r *Reader) List(ctx context.Context, filter Filter) (Page, error) {
	where, args, argIdx := buildWhere(r.user, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM items WHERE `+where, args...).Scan(&total); err != nil {
		return Page{}, fmt.Errorf("failed to count items: %w", err)
	}

	query := `SELECT ` + itemColumns + `, ` + tagNamesSubquery + ` FROM items WHERE ` + where
	query += ` ORDER BY items.captured_at DESC, items.id`
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, filter.Limit, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return Page{}, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	found, err := pgx.CollectRows(rows, scanItem)
	if err != nil {
		return Page{}, fmt.Errorf("failed to list items: %w", err)
	}

	return Page{
		Items:  found,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func buildWhere(user domain.UserID, filter Filter) (string, []any, int) {
	args := []any{user.String()}
	clauses := []string{"items.user_id = $1"}
	argIdx := 2

	if filter.Search != "" {
		clauses = append(clauses, fmt.Sprintf("items.body ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, filter.Search)
		argIdx++
	}

	if len(filter.Tags) > 0 {
		clauses = append(clauses, fmt.Sprintf(`items.id IN (
			SELECT it2.item_id
			FROM item_tags it2
			JOIN tags t2 ON t2.id = it2.tag_id
			WHERE t2.name = ANY($%d::text[])
			GROUP BY it2.item_id
			HAVING COUNT(DISTINCT t2.name) = $%d
		)`, argIdx, argIdx+1))
		args = append(args, filter.Tags, len(filter.Tags))
		argIdx += 2
	}

	if len(filter.States) > 0 {
		placeholders := make([]string, len(filter.States))
		for i, s := range filter.States {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, s)
			argIdx++
		}
		clauses = append(clauses, fmt.Sprintf("items.state IN (%s)", strings.Join(placeholders, ", ")))
	}

	return strings.Join(clauses, " AND "), args, argIdx
}

func (r *Reader) ByID(ctx context.Context, id string) (*Item, error) {
	var i Item
	var tagJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT `+itemColumns+`, `+tagNamesSubquery+`
		FROM items
		WHERE items.id = $1 AND items.user_id = $2
	`, id, r.user.String()).Scan(
		&i.ID, &i.Type, &i.Body, &i.State, &i.Version,
		&i.Title, &i.Description, &i.FilePath, &i.ThumbnailPath,
		&i.FileName, &i.FileSize, &i.MimeType, &i.CapturedAt,
		&tagJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read item %s: %w", id, err)
	}
	if len(tagJSON) > 0 {
		json.Unmarshal(tagJSON, &i.TagNames)
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
	var tagJSON []byte
	err := r.Scan(
		&i.ID, &i.Type, &i.Body, &i.State, &i.Version,
		&i.Title, &i.Description, &i.FilePath, &i.ThumbnailPath,
		&i.FileName, &i.FileSize, &i.MimeType, &i.CapturedAt,
		&tagJSON,
	)
	if err != nil {
		return i, err
	}
	if len(tagJSON) > 0 {
		json.Unmarshal(tagJSON, &i.TagNames)
	}
	return i, nil
}
