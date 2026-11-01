package item

import (
	"context"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
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

func (r *Repository) For(user domain.UserID) *Scoped {
	return &Scoped{
		events: r.events,
		user:   user,
	}
}

type Scoped struct {
	events *store.OfEvents
	user   domain.UserID
}

func (s *Scoped) Load(ctx context.Context, id uuid.UUID) (*Item, error) {
	return s.events.Load[Item](ctx, StreamID(s.user, id))
}

func (s *Scoped) Save(ctx context.Context, i *Item, causeID uuid.UUID) error {
	return s.events.Save(ctx, StreamID(s.user, i.ID()), i, causeID)
}
