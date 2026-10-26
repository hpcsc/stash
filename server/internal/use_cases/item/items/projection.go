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
		event.TextCaptured{}.EventType(),
		event.LinkCaptured{}.EventType(),
		event.LinkMetadataFetched{}.EventType(),
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
