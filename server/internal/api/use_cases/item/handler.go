package item

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gookit/validate"
	"github.com/hpcsc/stash/internal/api/middleware"
	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/route"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/command"
	"github.com/hpcsc/stash/internal/domain/event"
	"github.com/hpcsc/stash/internal/storage"
	itemCommand "github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/items"
	trashCommand "github.com/hpcsc/stash/internal/use_cases/trash/command"
	"github.com/unrolled/render"
)

type ItemReader interface {
	For(user domain.UserID) items.View
}

var _ route.Routable = (*handler)(nil)

func NewHandler(bus *command.Bus, items ItemReader, store storage.Storage, issuer *session.Issuer) route.Routable {
	return &handler{
		renderer: render.New(),
		bus:      bus,
		items:    items,
		storage:  store,
		issuer:   issuer,
	}
}

type handler struct {
	renderer *render.Render
	bus      *command.Bus
	items    ItemReader
	storage  storage.Storage
	issuer   *session.Issuer
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Protected("POST", "/items", h.capture),
		route.Protected("GET", "/items", h.list),
		route.Protected("GET", "/items/{id}", h.get),
		route.Protected("POST", "/items/{id}/keep", h.keep),
		route.Protected("POST", "/items/{id}/drop", h.dropItem),
		route.Protected("POST", "/items/{id}/restore", h.restoreItem),
		route.Protected("POST", "/items/{id}/delete", h.deleteItem),
		route.Protected("POST", "/trash/empty", h.emptyTrash),
		route.Public("GET", "/files/*", h.serveFile),
	}
}

func (h *handler) capture(w http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		h.captureUpload(w, req)
		return
	}

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
		_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(h.captureResponse(req, itemID, "link", body.Body)))
	default:
		if !h.dispatch(w, req, itemCommand.CaptureNote{
			ItemID:     itemID,
			Markdown:   body.Body,
			CapturedAt: now,
		}) {
			return
		}
		_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(NoteResponse{
			ID:         itemID.String(),
			Type:       "note",
			Body:       body.Body,
			State:      "inbox",
			CapturedAt: now,
		}))
	}
}

func (h *handler) view(req *http.Request) items.View {
	return h.items.For(middleware.UserFromContext(req.Context()))
}

func (h *handler) captureResponse(req *http.Request, itemID uuid.UUID, itemType, body string) CaptureResponse {
	resp := CaptureResponse{ID: itemID.String()}
	existing, err := h.view(req).Existing(req.Context(), itemType, body)
	if err != nil || existing == nil {
		return resp
	}
	resp.Duplicate = true
	resp.DuplicateState = existing.State
	return resp
}

func (h *handler) list(w http.ResponseWriter, req *http.Request) {
	search := req.URL.Query().Get("q")
	states := req.URL.Query()["state"]
	all, err := h.view(req).List(req.Context(), search, states...)
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
	i, err := h.view(req).ByID(req.Context(), chi.URLParam(req, "id"))
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

func (h *handler) restoreItem(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("invalid item id"))
		return
	}

	if !h.dispatch(w, req, itemCommand.Restore{ItemID: id}) {
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
}

func (h *handler) deleteItem(w http.ResponseWriter, req *http.Request) {
	id, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("invalid item id"))
		return
	}

	if !h.dispatch(w, req, itemCommand.Delete{ItemID: id}) {
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.Succeed())
}

func (h *handler) emptyTrash(w http.ResponseWriter, req *http.Request) {
	dropped, err := h.view(req).List(req.Context(), "", "dropped")
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}

	ids := make([]uuid.UUID, 0, len(dropped))
	for _, i := range dropped {
		id, err := uuid.Parse(i.ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("trash is empty"))
		return
	}

	if !h.dispatch(w, req, trashCommand.EmptyTrash{ItemIDs: ids}) {
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
	err := h.bus.Dispatch(req.Context(), domain.NewCommand(middleware.UserFromContext(req.Context()), payload))
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

const (
	maxNoteUploadSize  = itemCommand.MaxNoteSize + (1 << 20) // note images plus multipart overhead
	maxMultipartMemory = 32 << 20
)

func (h *handler) captureUpload(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, maxNoteUploadSize)

	if err := req.ParseMultipartForm(maxMultipartMemory); err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("upload is too large, max 100 MB"))
		return
	}

	_, hasMarkdown := req.MultipartForm.Value["markdown"]
	_, hasImages := req.MultipartForm.File["images"]
	if hasMarkdown || hasImages {
		h.captureNote(w, req)
		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("missing file field"))
		return
	}
	defer file.Close()

	if header.Size > itemCommand.MaxFileSize {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("file too large, max 25 MB"))
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if strings.HasPrefix(mimeType, "video/") {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("videos are not accepted"))
		return
	}

	itemID := uuid.New()
	fileName := filepath.Base(header.Filename)
	ext := filepath.Ext(fileName)
	now := time.Now()

	if isImage(mimeType) {
		key := domain.UserStream(middleware.UserFromContext(req.Context()), "images/"+itemID.String()+ext)
		if !h.saveUpload(w, req, key, file) {
			return
		}
		if !h.dispatch(w, req, itemCommand.CaptureImage{
			ItemID:     itemID,
			FileRef:    key,
			CapturedAt: now,
		}) {
			return
		}
		_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{
			"id":       itemID.String(),
			"type":     "image",
			"filePath": key,
		}))
		return
	}

	key := domain.UserStream(middleware.UserFromContext(req.Context()), "files/"+itemID.String()+ext)
	if !h.saveUpload(w, req, key, file) {
		return
	}
	if !h.dispatch(w, req, itemCommand.CaptureFile{
		ItemID:     itemID,
		FileRef:    key,
		FileName:   fileName,
		FileSize:   header.Size,
		MimeType:   mimeType,
		CapturedAt: now,
	}) {
		return
	}
	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(map[string]string{
		"id":       itemID.String(),
		"type":     "file",
		"filePath": key,
		"fileName": fileName,
	}))
}

func (h *handler) captureNote(w http.ResponseWriter, req *http.Request) {
	markdown := req.FormValue("markdown")
	headers := req.MultipartForm.File["images"]
	if len(headers) > itemCommand.MaxNoteImages {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(fmt.Sprintf("too many images, max %d", itemCommand.MaxNoteImages)))
		return
	}
	if strings.TrimSpace(markdown) == "" && len(headers) == 0 {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("note must have text or an image"))
		return
	}

	var totalSize int64
	for _, header := range headers {
		if header.Size > itemCommand.MaxFileSize {
			_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("image too large, max 25 MB"))
			return
		}
		totalSize += header.Size
		if totalSize > itemCommand.MaxNoteSize {
			_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("note is too large, max 100 MB"))
			return
		}

		mimeType := header.Header.Get("Content-Type")
		if strings.HasPrefix(mimeType, "video/") {
			_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("videos are not accepted"))
			return
		}
		if !isImage(mimeType) {
			_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("only images can be added to a note"))
			return
		}
	}

	itemID := uuid.New()
	now := time.Now()
	user := middleware.UserFromContext(req.Context())
	for idx, header := range headers {
		file, err := header.Open()
		if err != nil {
			_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("failed to read image"))
			return
		}

		imageID := uuid.New()
		fileName := filepath.Base(header.Filename)
		key := domain.UserStream(user, "images/"+imageID.String()+filepath.Ext(fileName))
		if !h.saveUpload(w, req, key, file) {
			file.Close()
			return
		}
		file.Close()

		markdown = strings.ReplaceAll(markdown, "@stash-image-"+strconv.Itoa(idx), key)
	}

	if !h.dispatch(w, req, itemCommand.CaptureNote{
		ItemID:     itemID,
		Markdown:   markdown,
		CapturedAt: now,
	}) {
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(NoteResponse{
		ID:         itemID.String(),
		Type:       "note",
		Body:       markdown,
		State:      "inbox",
		CapturedAt: now,
	}))
}

func (h *handler) saveUpload(w http.ResponseWriter, req *http.Request, key string, file io.Reader) bool {
	if _, err := h.storage.Save(req.Context(), key, file); err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail("failed to save file"))
		return false
	}
	return true
}

func isImage(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp":
		return true
	}
	return false
}

func (h *handler) serveFile(w http.ResponseWriter, req *http.Request) {
	user := h.fileSessionUser(req)
	if user == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(response.Fail("sign in to continue"))
		return
	}

	key := chi.URLParam(req, "*")
	owner, err := domain.UserOf(key)
	if err != nil || owner != user {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("file not found"))
		return
	}

	reader, err := h.storage.Open(req.Context(), key)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("file not found"))
		return
	}
	defer reader.Close()

	w.Header().Set("Cache-Control", "private, max-age=86400")
	if seeker, ok := reader.(io.ReadSeeker); ok {
		http.ServeContent(w, req, key, time.Now(), seeker)
		return
	}

	io.Copy(w, reader)
}

func (h *handler) fileSessionUser(req *http.Request) domain.UserID {
	token := req.URL.Query().Get("token")
	if token == "" {
		token = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	}

	subject, err := h.issuer.Verify(token)
	user := domain.UserID(subject)
	if err != nil || !user.Valid() {
		return ""
	}
	return user
}
