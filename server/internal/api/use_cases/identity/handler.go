package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gookit/validate"
	"github.com/hpcsc/stash/internal/api/middleware"
	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/route"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	identityCommand "github.com/hpcsc/stash/internal/use_cases/identity/command"
	"github.com/unrolled/render"
)

type IDTokenVerifier interface {
	Verify(ctx context.Context, raw string) (string, error)
}

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus, verifiers map[string]IDTokenVerifier, issuer *session.Issuer) route.Routable {
	return &handler{
		renderer:  render.New(),
		bus:       bus,
		verifiers: verifiers,
		issuer:    issuer,
	}
}

type handler struct {
	renderer  *render.Render
	bus       *command.Bus
	verifiers map[string]IDTokenVerifier
	issuer    *session.Issuer
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Public("POST", "/auth/signin", h.signIn),
		route.Protected("POST", "/auth/signout", h.signOut),
	}
}

func (h *handler) signIn(w http.ResponseWriter, req *http.Request) {
	var body SignInRequest
	if !h.decode(w, req, &body) {
		return
	}

	verifier, ok := h.verifiers[body.Provider]
	if !ok {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("unsupported provider"))
		return
	}

	subject, err := verifier.Verify(req.Context(), body.IDToken)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusUnauthorized, response.Fail("sign-in failed"))
		return
	}

	if !h.dispatch(w, req, identityCommand.SignIn{Subject: subject, Provider: body.Provider}) {
		return
	}

	token, err := h.issuer.Issue(subject)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail("failed to issue session"))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{"token": token}))
}

func (h *handler) signOut(w http.ResponseWriter, req *http.Request) {
	subject := middleware.SubjectFromContext(req.Context())
	if !h.dispatch(w, req, identityCommand.SignOut{Subject: subject}) {
		return
	}
	_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
}

func (h *handler) decode(w http.ResponseWriter, req *http.Request, body any) bool {
	if err := json.NewDecoder(req.Body).Decode(body); err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("received invalid request body"))
		return false
	}

	v := validate.Struct(body)
	if !v.Validate() {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.FailWithValidationErrors(v.Errors))
		return false
	}
	return true
}

func (h *handler) dispatch[P any](w http.ResponseWriter, req *http.Request, payload P) bool {
	err := h.bus.Dispatch(req.Context(), domain.NewCommand(payload))
	if errors.Is(err, event.ErrConcurrencyConflict) {
		_ = h.renderer.JSON(w, http.StatusConflict, response.Fail(fmt.Sprintf("%v, retry the request", err)))
		return false
	}
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return false
	}
	return true
}
