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
	apiIdentity "github.com/hpcsc/stash/internal/api/use_cases/identity"
	apiItem "github.com/hpcsc/stash/internal/api/use_cases/item"
	"github.com/hpcsc/stash/internal/api/use_cases/root"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/domain/event/stream"
	coreUseCases "github.com/hpcsc/stash/internal/use_cases"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(router chi.Router, cfg *api.Config, pool *pgxpool.Pool, logger *slog.Logger) error {
	if cfg.SessionSecret == "" {
		return errors.New("SESSION_SECRET must be set")
	}
	if cfg.GoogleClientID == "" {
		return errors.New("GOOGLE_CLIENT_ID must be set")
	}

	issuer := session.New([]byte(cfg.SessionSecret), session.DefaultTTL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	googleVerifier, err := google.New(ctx, cfg.GoogleClientID)
	if err != nil {
		return err
	}

	authMiddleware := middleware.NewAuthMiddleware(issuer)

	verifiers := map[string]apiIdentity.IDTokenVerifier{"google": googleVerifier}

	var publicRoutes, protectedRoutes []*route.Route
	for _, r := range allRoutes(pool, issuer, verifiers) {
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

	return nil
}

func allRoutes(pool *pgxpool.Pool, issuer *session.Issuer, verifiers map[string]apiIdentity.IDTokenVerifier) []*route.Route {
	reg := registry.New()
	coreUseCases.RegisterEvents(reg)

	bus := command.NewBus()
	coreUseCases.RegisterHandlers(bus, stream.NewPostgres(pool, reg))

	routables := []route.Routable{
		root.NewHandler(),
		apiIdentity.NewHandler(bus, verifiers, issuer),
		apiItem.NewHandler(bus, items.NewQuery(pool)),
	}

	var routes []*route.Route
	for _, r := range routables {
		routes = append(routes, r.Routes()...)
	}
	return routes
}
