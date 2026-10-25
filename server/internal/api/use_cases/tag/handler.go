package tag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gookit/validate"
	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/route"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	tagCommand "github.com/hpcsc/stash/internal/use_cases/tag/command"
	"github.com/hpcsc/stash/internal/use_cases/tag/tags"
	"github.com/unrolled/render"
)

type TagReader interface {
	List(ctx context.Context) ([]tags.Tag, error)
	ByID(ctx context.Context, id string) (*tags.Tag, error)
}

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus, tags TagReader) route.Routable {
	return &handler{
		renderer: render.New(),
		bus:      bus,
		tags:     tags,
	}
}

type handler struct {
	renderer *render.Render
	bus      *command.Bus
	tags     TagReader
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Protected("POST", "/tags", h.create),
		route.Protected("GET", "/tags", h.list),
		route.Protected("POST", "/tags/{id}/rename", h.rename),
	}
}

func (h *handler) create(w http.ResponseWriter, req *http.Request) {
	var body createRequest
	if !h.decode(w, req, &body) {
		return
	}

	tagID := uuid.New()
	if !h.dispatch(w, req, tagCommand.CreateTag{
		TagID: tagID,
		Name:  body.Name,
	}) {
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{"id": tagID.String()}))
}

func (h *handler) list(w http.ResponseWriter, req *http.Request) {
	all, err := h.tags.List(req.Context())
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if all == nil {
		all = []tags.Tag{}
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(all))
}

func (h *handler) rename(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("invalid tag id"))
		return
	}

	var body renameRequest
	if !h.decode(w, req, &body) {
		return
	}

	if !h.dispatch(w, req, tagCommand.RenameTag{TagID: id, Name: body.Name}) {
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

type createRequest struct {
	Name string `json:"name" validate:"required"`
}

type renameRequest struct {
	Name string `json:"name" validate:"required"`
}