package trash

import (
	"context"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/domain/event/store"
)

func StreamID(user domain.UserID) string {
	return domain.UserStream(user, "trash")
}

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

func (s *Scoped) Load(ctx context.Context) (*Trash, error) {
	return s.events.Load[Trash](ctx, StreamID(s.user))
}

func (s *Scoped) Save(ctx context.Context, t *Trash, causeID uuid.UUID) error {
	return s.events.Save(ctx, StreamID(s.user), t, causeID)
}
