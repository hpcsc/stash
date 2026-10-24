package item

import (
	"context"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/store"
)

func NewRepository(stream event.Stream) *Repository {
	return &Repository{
		events: store.New(stream),
	}
}

type Repository struct {
	events *store.OfEvents
}

func (r *Repository) Load(ctx context.Context, id uuid.UUID) (*Item, error) {
	return r.events.Load[Item](ctx, StreamID(id))
}

func (r *Repository) Save(ctx context.Context, i *Item, causeID uuid.UUID) error {
	return r.events.Save(ctx, StreamID(i.ID()), i, causeID)
}
