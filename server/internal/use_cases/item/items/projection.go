package items

import (
	"context"
	"encoding/json"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
	"github.com/hpcsc/stash/internal/use_cases/item/note"
	"github.com/jackc/pgx/v5"
)

func NewProjection() *Projection {
	return &Projection{}
}

type Projection struct{}

func (p *Projection) Name() string {
	return "items"
}

func (p *Projection) EventTypes() []string {
	return []string{
		event.TextCaptured{}.EventType(),
		event.LinkCaptured{}.EventType(),
		event.ImageCaptured{}.EventType(),
		event.FileCaptured{}.EventType(),
		event.NoteCaptured{}.EventType(),
		event.LinkMetadataFetched{}.EventType(),
		event.ThumbnailGenerated{}.EventType(),
		event.ImageThumbnailGenerated{}.EventType(),
		event.ItemKept{}.EventType(),
		event.ItemDropped{}.EventType(),
	}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *event.TextCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, type, body, source_app, source_url, captured_at, state, version)
			VALUES ($1, 'text', $2, $3, $4, $5, 'inbox', $6)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, payload.Body, payload.SourceApp, payload.SourceURL, payload.CapturedAt, evt.Version)
		return err
	case *event.LinkCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, type, body, captured_at, state, version)
			VALUES ($1, 'link', $2, $3, 'inbox', $4)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, payload.URL, payload.CapturedAt, evt.Version)
		return err
	case *event.LinkMetadataFetched:
		_, err := tx.Exec(ctx, `
			UPDATE items SET title = $2, description = $3, version = $4, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, payload.Title, payload.Description, evt.Version)
		return err
	case *event.ImageCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, type, body, file_path, captured_at, state, version)
			VALUES ($1, 'image', '', $2, $3, 'inbox', $4)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, payload.FileRef, payload.CapturedAt, evt.Version)
		return err
	case *event.ThumbnailGenerated:
		_, err := tx.Exec(ctx, `
			UPDATE items SET thumbnail_path = $2, version = $3, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, payload.ThumbnailRef, evt.Version)
		return err
	case *event.ImageThumbnailGenerated:
		_, err := tx.Exec(ctx, `
			UPDATE item_images SET thumbnail_path = $2 WHERE id = $1
		`, payload.ImageID, payload.ThumbnailRef)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE items SET version = $2, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, evt.Version)
		return err
	case *event.FileCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, type, body, file_path, file_name, file_size, mime_type, captured_at, state, version)
			VALUES ($1, 'file', '', $2, $3, $4, $5, $6, 'inbox', $7)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, payload.FileRef, payload.FileName, payload.FileSize, payload.MimeType, payload.CapturedAt, evt.Version)
		return err
	case *event.NoteCaptured:
		blocks, err := json.Marshal(blocksForStorage(payload.Blocks))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO items (id, type, body, note_blocks, captured_at, state, version)
			VALUES ($1, 'note', $2, $3, $4, 'inbox', $5)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, note.PlainText(payload.Blocks), blocks, payload.CapturedAt, evt.Version)
		if err != nil {
			return err
		}

		sortOrder := 0
		for _, b := range payload.Blocks {
			if b.Image == nil {
				continue
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO item_images (id, item_id, file_path, file_name, file_size, mime_type, sort_order)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (id) DO NOTHING
			`, b.Image.ID, payload.ItemID, b.Image.FileRef, b.Image.FileName, b.Image.FileSize, b.Image.MimeType, sortOrder)
			if err != nil {
				return err
			}
			sortOrder++
		}
		return nil
	case *event.ItemKept:
		_, err := tx.Exec(ctx, `
			UPDATE items SET state = 'kept', version = $2, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, evt.Version)
		return err
	case *event.ItemDropped:
		_, err := tx.Exec(ctx, `
			UPDATE items SET state = 'dropped', version = $2, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, evt.Version)
		return err
	}
	return nil
}

type storedBlock struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	ImageID string `json:"imageId,omitempty"`
}

func blocksForStorage(blocks []note.Block) []storedBlock {
	stored := make([]storedBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.Image != nil {
			stored = append(stored, storedBlock{Type: note.TypeImage, ImageID: b.Image.ID.String()})
			continue
		}
		stored = append(stored, storedBlock{Type: note.TypeText, Text: b.Text})
	}
	return stored
}
