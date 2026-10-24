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
	return []string{event.TextCaptured{}.EventType()}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *event.TextCaptured:
		_, err := tx.Exec(ctx, `
			INSERT INTO items (id, type, body, source_app, source_url, captured_at, version)
			VALUES ($1, 'text', $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING
		`, payload.ItemID, payload.Body, payload.SourceApp, payload.SourceURL, payload.CapturedAt, evt.Version)
		return err
	}
	return nil
}
