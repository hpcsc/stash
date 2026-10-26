package use_cases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/stash/internal/api"
	"github.com/hpcsc/stash/internal/api/google"
	"github.com/hpcsc/stash/internal/api/middleware"
	"github.com/hpcsc/stash/internal/api/route"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/hpcsc/stash/internal/api/websocket"
	apiIdentity "github.com/hpcsc/stash/internal/api/use_cases/identity"
	apiItem "github.com/hpcsc/stash/internal/api/use_cases/item"
	apiTag "github.com/hpcsc/stash/internal/api/use_cases/tag"
	"github.com/hpcsc/stash/internal/api/use_cases/root"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/domain/event/stream"
	"github.com/hpcsc/stash/internal/domain/event/subscription"
	"github.com/hpcsc/stash/internal/storage"
	coreUseCases "github.com/hpcsc/stash/internal/use_cases"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	"github.com/hpcsc/stash/internal/use_cases/tag/tags"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(router chi.Router, cfg *api.Config, pool *pgxpool.Pool, logger *slog.Logger) (func(), error) {
	if cfg.SessionSecret == "" {
		return nil, errors.New("SESSION_SECRET must be set")
	}
	if cfg.GoogleClientID == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID must be set")
	}

	issuer := session.New([]byte(cfg.SessionSecret), session.DefaultTTL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	googleVerifier, err := google.New(ctx, cfg.GoogleClientID)
	if err != nil {
		return nil, err
	}

	authMiddleware := middleware.NewAuthMiddleware(issuer)

	verifiers := map[string]apiIdentity.IDTokenVerifier{"google": googleVerifier}

	reg := registry.New()
	coreUseCases.RegisterEvents(reg)

	store, err := storage.NewDisk(cfg.StorageDir)
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}

	wsHub := websocket.NewHub(logger)

	subCtx, cancelSub := context.WithCancel(context.Background())
	go subscribeToPushEvents(subCtx, pool, reg, wsHub, logger)

	cleanup := func() {
		cancelSub()
		wsHub.Shutdown(context.Background())
	}

	var publicRoutes, protectedRoutes []*route.Route
	for _, r := range allRoutes(pool, issuer, verifiers, reg, wsHub, store) {
		if r.IsPublic() {
			publicRoutes = append(publicRoutes, r)
		} else if r.IsProtected() {
			protectedRoutes = append(protectedRoutes, r)
		}
	}

	for _, r := range publicRoutes {
		router.MethodFunc(r.Method, r.Pattern, r.Handler)
		logger.Info(fmt.Sprintf("registered public route %s %s", r.Method, r.Pattern))
	}

	router.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(authMiddleware)

		for _, r := range protectedRoutes {
			protectedRouter.MethodFunc(r.Method, r.Pattern, r.Handler)
			logger.Info(fmt.Sprintf("registered protected route %s %s", r.Method, r.Pattern))
		}
	})

	return cleanup, nil
}

func allRoutes(pool *pgxpool.Pool, issuer *session.Issuer, verifiers map[string]apiIdentity.IDTokenVerifier, reg *registry.OfEvents, wsHub *websocket.Hub, store storage.Storage) []*route.Route {
	bus := command.NewBus()
	coreUseCases.RegisterHandlers(bus, stream.NewPostgres(pool, reg))

	routables := []route.Routable{
		root.NewHandler(),
		apiIdentity.NewHandler(bus, verifiers, issuer),
		apiItem.NewHandler(bus, items.NewQuery(pool), store),
		apiTag.NewHandler(bus, tags.NewQuery(pool)),
		websocket.NewHandler(wsHub, issuer, slog.Default()),
	}

	var routes []*route.Route
	for _, r := range routables {
		routes = append(routes, r.Routes()...)
	}
	return routes
}

func subscribeToPushEvents(ctx context.Context, pool *pgxpool.Pool, reg *registry.OfEvents, hub *websocket.Hub, logger *slog.Logger) {
	sub := subscription.NewPostgres(pool, reg, logger)

	after, err := sub.LastPosition(ctx)
	if err != nil {
		logger.Error("failed to get last position for push subscription", "error", err)
		return
	}

	eventTypes := []string{
		event.LinkMetadataFetched{}.EventType(),
		event.ThumbnailGenerated{}.EventType(),
	}
	events, err := sub.SubscribeToAll(ctx, after, eventTypes)
	if err != nil {
		logger.Error("failed to subscribe to push events", "error", err)
		return
	}

	for evt := range events {
		switch payload := evt.Payload.(type) {
		case *event.LinkMetadataFetched:
			hub.Broadcast(context.Background(), websocket.Message{
				Type: "link-metadata-fetched",
				Data: map[string]string{
					"id":          payload.ItemID.String(),
					"title":       payload.Title,
					"description": payload.Description,
				},
			})
		case *event.ThumbnailGenerated:
			hub.Broadcast(context.Background(), websocket.Message{
				Type: "thumbnail-generated",
				Data: map[string]string{
					"id":            payload.ItemID.String(),
					"thumbnailPath": payload.ThumbnailRef,
				},
			})
		}
	}

	logger.Info("push event subscription stopped")
}
