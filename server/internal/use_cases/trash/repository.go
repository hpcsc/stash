package trash

import (
	"context"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/store"
)

const StreamID = "trash"

func NewRepository(stream event.Stream) *Repository {
	return &Repository{
		events: store.New(stream),
	}
}

type Repository struct {
	events *store.OfEvents
}

func (r *Repository) Load(ctx context.Context) (*Trash, error) {
	return r.events.Load[Trash](ctx, StreamID)
}

func (r *Repository) Save(ctx context.Context, t *Trash, causeID uuid.UUID) error {
	return r.events.Save(ctx, StreamID, t, causeID)
}
