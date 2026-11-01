package items

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
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
		event.LinkCaptured{}.EventType(),
		event.ImageCaptured{}.EventType(),
		event.FileCaptured{}.EventType(),
		event.NoteCaptured{}.EventType(),
		event.LinkMetadataFetched{}.EventType(),
		event.ThumbnailGenerated{}.EventType(),
		event.ItemKept{}.EventType(),
		event.ItemDropped{}.EventType(),
		event.ItemRestored{}.EventType(),
		event.ItemDeleted{}.EventType(),
	}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	user, err := domain.UserOf(evt.StreamID)
	if err != nil {
		return err
	}

	switch payload := evt.Payload.(type) {
	case *event.LinkCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, user_id, type, body, captured_at, state, version)
			VALUES ($1, $2, 'link', $3, $4, 'inbox', $5)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, user.String(), payload.URL, payload.CapturedAt, evt.Version)
		return err
	case *event.LinkMetadataFetched:
		_, err := tx.Exec(ctx, `
			UPDATE items SET title = $2, description = $3, version = $4, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, payload.Title, payload.Description, evt.Version)
		return err
	case *event.ImageCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, user_id, type, body, file_path, captured_at, state, version)
			VALUES ($1, $2, 'image', '', $3, $4, 'inbox', $5)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, user.String(), payload.FileRef, payload.CapturedAt, evt.Version)
		return err
	case *event.ThumbnailGenerated:
		_, err := tx.Exec(ctx, `
			UPDATE items SET thumbnail_path = $2, version = $3, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, payload.ThumbnailRef, evt.Version)
		return err
	case *event.FileCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, user_id, type, body, file_path, file_name, file_size, mime_type, captured_at, state, version)
			VALUES ($1, $2, 'file', '', $3, $4, $5, $6, $7, 'inbox', $8)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, user.String(), payload.FileRef, payload.FileName, payload.FileSize, payload.MimeType, payload.CapturedAt, evt.Version)
		return err
	case *event.NoteCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, user_id, type, body, captured_at, state, version)
			VALUES ($1, $2, 'note', $3, $4, 'inbox', $5)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, user.String(), payload.Markdown, payload.CapturedAt, evt.Version)
		return err
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
	case *event.ItemRestored:
		state := "inbox"
		if payload.RestoredTo == "library" {
			state = "kept"
		}
		_, err := tx.Exec(ctx, `
			UPDATE items SET state = $2, version = $3, updated_at = NOW()
			WHERE id = $1
		`, payload.ItemID, state, evt.Version)
		return err
	case *event.ItemDeleted:
		_, err := tx.Exec(ctx, `DELETE FROM items WHERE id = $1`, payload.ItemID)
		return err
	}
	return nil
}
