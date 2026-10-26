package item

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gookit/validate"
	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/route"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	"github.com/unrolled/render"
)

type ItemReader interface {
	List(ctx context.Context, search string, states ...string) ([]items.Item, error)
	ByID(ctx context.Context, id string) (*items.Item, error)
}

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus, items ItemReader) route.Routable {
	return &handler{
		renderer: render.New(),
		bus:      bus,
		items:    items,
	}
}

type handler struct {
	renderer *render.Render
	bus      *command.Bus
	items    ItemReader
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Protected("POST", "/items", h.capture),
		route.Protected("GET", "/items", h.list),
		route.Protected("GET", "/items/{id}", h.get),
		route.Protected("POST", "/items/{id}/keep", h.keep),
		route.Protected("POST", "/items/{id}/drop", h.dropItem),
	}
}

func (h *handler) capture(w http.ResponseWriter, req *http.Request) {
	var body CaptureRequest
	if !h.decode(w, req, &body) {
		return
	}

	itemID := uuid.New()
	now := time.Now()

	switch body.Type {
	case "link":
		if !h.dispatch(w, req, itemCommand.CaptureLink{
			ItemID:     itemID,
			URL:        body.Body,
			CapturedAt: now,
		}) {
			return
		}
		_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{"id": itemID.String()}))
	default:
		if !h.dispatch(w, req, itemCommand.CaptureText{
			ItemID:     itemID,
			Body:       body.Body,
			SourceApp:  body.SourceApp,
			SourceURL:  body.SourceURL,
			CapturedAt: now,
		}) {
			return
		}
		_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{"id": itemID.String()}))
	}
}

func (h *handler) list(w http.ResponseWriter, req *http.Request) {
	search := req.URL.Query().Get("q")
	states := req.URL.Query()["state"]
	all, err := h.items.List(req.Context(), search, states...)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if all == nil {
		all = []items.Item{}
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(all))
}

func (h *handler) get(w http.ResponseWriter, req *http.Request) {
	i, err := h.items.ByID(req.Context(), chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if i == nil {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("item not found"))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(i))
}

func (h *handler) keep(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("invalid item id"))
		return
	}

	var body KeepRequest
	_ = json.NewDecoder(req.Body).Decode(&body)

	if !h.dispatch(w, req, itemCommand.Keep{ItemID: id, TagIDs: parseTagIDs(body.Tags)}) {
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
}

func (h *handler) dropItem(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("invalid item id"))
		return
	}

	if !h.dispatch(w, req, itemCommand.Drop{ItemID: id}) {
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