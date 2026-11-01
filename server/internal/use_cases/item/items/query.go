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

type Item struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Body          string    `json:"body"`
	State         string    `json:"state"`
	SourceApp     string    `json:"sourceApp,omitempty"`
	SourceURL     string    `json:"sourceUrl,omitempty"`
	Title         string    `json:"title,omitempty"`
	Description   string    `json:"description,omitempty"`
	FilePath      string    `json:"filePath,omitempty"`
	ThumbnailPath string    `json:"thumbnailPath,omitempty"`
	FileName      string    `json:"fileName,omitempty"`
	FileSize      int64     `json:"fileSize,omitempty"`
	MimeType      string    `json:"mimeType,omitempty"`
	CapturedAt    time.Time `json:"capturedAt"`
	Images        []Image   `json:"images,omitempty"`
	Blocks        []Block   `json:"blocks,omitempty"`

	stored []storedBlock
}

type Image struct {
	ID            string `json:"id"`
	FilePath      string `json:"filePath"`
	ThumbnailPath string `json:"thumbnailPath,omitempty"`
	FileName      string `json:"fileName,omitempty"`
	FileSize      int64  `json:"fileSize,omitempty"`
	MimeType      string `json:"mimeType,omitempty"`
}

type Block struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Image *Image `json:"image,omitempty"`
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

func (r *Reader) List(ctx context.Context, search string, states ...string) ([]Item, error) {
	query := `
		SELECT id::text, type, body, state, source_app, source_url, title, description, file_path, thumbnail_path, file_name, file_size, mime_type, captured_at, note_blocks
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

	list, err := pgx.CollectRows(rows, scanItem)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	if err := r.attachImages(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Reader) ByID(ctx context.Context, id string) (*Item, error) {
	var i Item
	var blocks json.RawMessage
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, type, body, state, source_app, source_url, title, description, file_path, thumbnail_path, file_name, file_size, mime_type, captured_at, note_blocks
		FROM items
		WHERE id = $1 AND user_id = $2
	`, id, r.user.String()).Scan(&i.ID, &i.Type, &i.Body, &i.State, &i.SourceApp, &i.SourceURL, &i.Title, &i.Description, &i.FilePath, &i.ThumbnailPath, &i.FileName, &i.FileSize, &i.MimeType, &i.CapturedAt, &blocks)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read item %s: %w", id, err)
	}
	if i.Type == "note" {
		if err := i.readNoteBlocks(blocks); err != nil {
			return nil, err
		}
	}

	list := []Item{i}
	if err := r.attachImages(ctx, list); err != nil {
		return nil, err
	}
	return &list[0], nil
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
	var blocks json.RawMessage
	err := r.Scan(&i.ID, &i.Type, &i.Body, &i.State, &i.SourceApp, &i.SourceURL, &i.Title, &i.Description, &i.FilePath, &i.ThumbnailPath, &i.FileName, &i.FileSize, &i.MimeType, &i.CapturedAt, &blocks)
	if err != nil {
		return i, err
	}
	if i.Type == "note" {
		if err := i.readNoteBlocks(blocks); err != nil {
			return i, err
		}
	}
	return i, nil
}

func (i *Item) readNoteBlocks(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &i.stored); err != nil {
		return fmt.Errorf("failed to read note blocks for item %s: %w", i.ID, err)
	}
	return nil
}

func (r *Reader) attachImages(ctx context.Context, list []Item) error {
	ids := make([]string, 0, len(list))
	for _, i := range list {
		if i.Type == "note" {
			ids = append(ids, i.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id::text, item_id::text, file_path, COALESCE(thumbnail_path, ''), file_name, file_size, mime_type
		FROM item_images
		WHERE item_id::text = ANY($1)
		ORDER BY item_id, sort_order
	`, ids)
	if err != nil {
		return fmt.Errorf("failed to read note images: %w", err)
	}
	defer rows.Close()

	byItem := make(map[string][]Image, len(ids))
	for rows.Next() {
		var image Image
		var itemID string
		if err := rows.Scan(&image.ID, &itemID, &image.FilePath, &image.ThumbnailPath, &image.FileName, &image.FileSize, &image.MimeType); err != nil {
			return fmt.Errorf("failed to read note images: %w", err)
		}
		byItem[itemID] = append(byItem[itemID], image)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read note images: %w", err)
	}

	for idx := range list {
		if list[idx].Type != "note" {
			continue
		}
		list[idx].Images = byItem[list[idx].ID]
		list[idx].Blocks = blocksWithImages(list[idx].stored, list[idx].Images)
	}
	return nil
}

func blocksWithImages(stored []storedBlock, images []Image) []Block {
	byID := make(map[string]Image, len(images))
	for _, image := range images {
		byID[image.ID] = image
	}

	blocks := make([]Block, 0, len(stored))
	for _, b := range stored {
		if b.Type == "image" {
			image, ok := byID[b.ImageID]
			if !ok {
				continue
			}
			blocks = append(blocks, Block{Type: "image", Image: &image})
			continue
		}
		blocks = append(blocks, Block{Type: "text", Text: b.Text})
	}
	return blocks
}
