package tags

import (
	"context"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/tag/event"
	"github.com/jackc/pgx/v5"
)

func NewProjection() *Projection {
	return &Projection{}
}

type Projection struct{}

func (p *Projection) Name() string {
	return "tags"
}

func (p *Projection) EventTypes() []string {
	return []string{
		event.TagCreated{}.EventType(),
		event.TagRenamed{}.EventType(),
	}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	user, err := domain.UserOf(evt.StreamID)
	if err != nil {
		return err
	}

	switch payload := evt.Payload.(type) {
	case *event.TagCreated:
		_, err := tx.Exec(ctx, `
			INSERT INTO tags (id, user_id, name, version)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, payload.TagID, user.String(), payload.Name, evt.Version)
		return err
	case *event.TagRenamed:
		_, err := tx.Exec(ctx, `
			UPDATE tags SET name = $2, version = $3, updated_at = NOW()
			WHERE id = $1
		`, payload.TagID, payload.Name, evt.Version)
		return err
	}
	return nil
}
