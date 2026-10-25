package itemtags

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
	return "item-tags"
}

func (p *Projection) EventTypes() []string {
	return []string{event.ItemTagged{}.EventType()}
}

func (p *Projection) Handle(ctx context.Context, tx pgx.Tx, evt *domain.Event) error {
	switch payload := evt.Payload.(type) {
	case *event.ItemTagged:
		for _, tagID := range payload.TagIDs {
			_, err := tx.Exec(ctx, `
				INSERT INTO item_tags (item_id, tag_id)
				VALUES ($1, $2)
				ON CONFLICT (item_id, tag_id) DO NOTHING
			`, payload.ItemID, tagID)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}