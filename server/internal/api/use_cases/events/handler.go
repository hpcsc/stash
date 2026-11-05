package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/api/response"
	"github.com/hpcsc/stash/internal/api/route"
	"github.com/hpcsc/stash/internal/api/session"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/event/registry"
	"github.com/hpcsc/stash/internal/use_cases/consumers"
	"github.com/unrolled/render"
)

var _ route.Routable = (*handler)(nil)

type ReaderSource interface {
	Scoped(scope Scope) Reader
}

func NewHandler(source ReaderSource, reg *registry.OfEvents, issuer *session.Issuer, eventBrowserUserIDs []string, logger *slog.Logger) route.Routable {
	return &handler{
		renderer:          render.New(),
		source:            source,
		registry:          reg,
		issuer:            issuer,
		eventBrowserUsers: userIDSet(eventBrowserUserIDs),
		consumers:         consumerCatalog(),
		logger:            logger,
	}
}

type handler struct {
	renderer          *render.Render
	source            ReaderSource
	registry          *registry.OfEvents
	issuer            *session.Issuer
	eventBrowserUsers map[string]struct{}
	consumers         map[string]consumers.Description
	logger            *slog.Logger
}

func (h *handler) Routes() []*route.Route {
	return []*route.Route{
		route.Public("GET", "/events", h.requireScope(h.list)),
		route.Public("GET", "/events/types", h.requireScope(h.types)),
		route.Public("GET", "/events/streams", h.requireScope(h.streams)),
		route.Public("GET", "/events/checkpoints", h.requireScope(h.checkpoints)),
		route.Public("GET", "/events/projection", h.requireScope(h.projection)),
		route.Public("GET", "/events/stream", h.requireScope(h.streamEvents)),
		route.Public("GET", "/events/tail", h.tail),
	}
}

func (h *handler) requireScope(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, ok := resolveScope(r, h.issuer, h.eventBrowserUsers)
		if !ok {
			h.logger.Warn("request rejected, no valid session", "path", r.URL.Path)
			writeUnauthorized(w)
			return
		}
		if scope.IsAll() {
			h.logger.Info("served request with all-stream scope", "path", r.URL.Path)
		}
		next(w, r.WithContext(WithScope(r.Context(), scope)))
	}
}

func (h *handler) reader(w http.ResponseWriter, r *http.Request) (Reader, bool) {
	scope, ok := ScopeFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return nil, false
	}
	return h.source.Scoped(scope), true
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.reader(w, r)
	if !ok {
		return
	}

	filter, err := parseListFilter(r)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}

	stored, next, err := reader.List(r.Context(), filter)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(Page{
		Events: h.toEvents(stored),
		Next:   toPosition(next),
	}))
}

func (h *handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.reader(w, r)
	if !ok {
		return
	}

	streamID := r.URL.Query().Get("streamId")
	if streamID == "" {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("streamId is required"))
		return
	}

	stored, err := reader.Stream(r.Context(), streamID)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if len(stored) == 0 {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("stream not found"))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(h.toEvents(stored)))
}

func (h *handler) types(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.reader(w, r)
	if !ok {
		return
	}

	types, err := reader.Types(r.Context())
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if types == nil {
		types = []string{}
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(types))
}

func (h *handler) streams(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.reader(w, r)
	if !ok {
		return
	}

	streams, err := reader.Streams(r.Context())
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}
	if streams == nil {
		streams = []StreamSummary{}
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(streams))
}

func (h *handler) checkpoints(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.reader(w, r)
	if !ok {
		return
	}

	rows, head, err := reader.Checkpoints(r.Context())
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}

	checkpoints := make([]Checkpoint, 0, len(rows))
	for _, row := range rows {
		cp, err := h.checkpoint(r.Context(), reader, row, head)
		if err != nil {
			_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
			return
		}
		checkpoints = append(checkpoints, cp)
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(Checkpoints{
		LastPosition: toPosition(head),
		Checkpoints:  checkpoints,
	}))
}

func (h *handler) checkpoint(ctx context.Context, reader Reader, row CheckpointRow, head *domain.Position) (Checkpoint, error) {
	cp := Checkpoint{
		Name:      row.Name,
		Position:  Position{TransactionID: row.Position.TransactionID, Sequence: row.Position.Sequence},
		UpdatedAt: row.UpdatedAt,
	}

	description, known := h.consumers[row.Name]
	if !known {
		cp.Lag = lagFromHead(row.Position, head)
		return cp, nil
	}

	cp.Kind = string(description.Kind)
	cp.EventTypes = description.EventTypes

	lag, err := reader.CountAfter(ctx, row.Position, description.EventTypes)
	if err != nil {
		return Checkpoint{}, err
	}
	cp.Lag = lag

	next, err := reader.NextEvent(ctx, row.Position, description.EventTypes)
	if err != nil {
		return Checkpoint{}, err
	}
	cp.Next = next
	return cp, nil
}

func (h *handler) projection(w http.ResponseWriter, r *http.Request) {
	reader, ok := h.reader(w, r)
	if !ok {
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail("name is required"))
		return
	}

	filter, err := projectionFilter(r)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}

	table, err := reader.Projection(r.Context(), name, filter)
	if errors.Is(err, ErrUnknownProjection) {
		_ = h.renderer.JSON(w, http.StatusNotFound, response.Fail("unknown projection"))
		return
	}
	if errors.Is(err, ErrUnknownColumn) || errors.Is(err, ErrUnknownOperator) {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusInternalServerError, response.Fail(err.Error()))
		return
	}

	_ = h.renderer.JSON(w, http.StatusOK, response.SucceedWithData(table))
}

func projectionFilter(r *http.Request) (*ProjectionFilter, error) {
	values := r.URL.Query()
	column := strings.TrimSpace(values.Get("filterColumn"))
	if column == "" {
		return nil, nil
	}

	operator := ProjectionOperator(strings.TrimSpace(values.Get("filterOp")))
	if operator == "" {
		operator = ProjectionContains
	}
	if !operator.Valid() {
		return nil, fmt.Errorf("%w: %s", ErrUnknownOperator, operator)
	}

	return &ProjectionFilter{Column: column, Operator: operator, Value: values.Get("filterValue")}, nil
}

func lagFromHead(position domain.Position, head *domain.Position) uint64 {
	if head == nil || position.Sequence >= head.Sequence {
		return 0
	}
	return head.Sequence - position.Sequence
}

func consumerCatalog() map[string]consumers.Description {
	catalog := make(map[string]consumers.Description, len(consumers.Descriptions()))
	for _, description := range consumers.Descriptions() {
		catalog[description.Name] = description
	}
	return catalog
}

func (h *handler) tail(w http.ResponseWriter, r *http.Request) {
	scope, ok := resolveScope(r, h.issuer, h.eventBrowserUsers)
	if !ok {
		writeUnauthorized(w)
		return
	}
	if scope.IsAll() {
		h.logger.Info("served request with all-stream scope", "path", r.URL.Path)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	after, err := parseAfter(r)
	if err != nil {
		_ = h.renderer.JSON(w, http.StatusBadRequest, response.Fail(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	reader := h.source.Scoped(scope)
	err = reader.Tail(r.Context(), after, func(stored StoredEvent) error {
		data, marshalErr := json.Marshal(h.toEvent(stored))
		if marshalErr != nil {
			return marshalErr
		}
		if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", data); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})
	if err != nil && r.Context().Err() == nil {
		h.logger.Warn("debug stream ended", "error", err)
	}
}

func (h *handler) toEvents(stored []StoredEvent) []Event {
	events := make([]Event, 0, len(stored))
	for _, s := range stored {
		events = append(events, h.toEvent(s))
	}
	return events
}

func (h *handler) toEvent(stored StoredEvent) Event {
	event := Event{
		ID:            stored.ID,
		StreamID:      stored.StreamID,
		Version:       stored.Version,
		Position:      Position{TransactionID: stored.Position.TransactionID, Sequence: stored.Position.Sequence},
		Type:          stored.Type,
		SchemaVersion: stored.SchemaVersion,
		CreatedAt:     stored.CreatedAt,
	}
	if len(stored.Data) > 0 {
		event.Data = json.RawMessage(stored.Data)
	}
	if len(stored.Metadata) > 0 {
		event.Metadata = json.RawMessage(stored.Metadata)
	}

	payload, metadata, err := h.registry.ParseRawData(stored.Type, stored.SchemaVersion, stored.Data, stored.Metadata)
	if err != nil {
		event.ParseError = err.Error()
		return event
	}

	if decoded, marshalErr := json.Marshal(payload); marshalErr == nil {
		event.Decoded = json.RawMessage(decoded)
	}
	if metadata != nil {
		if metadata.CorrelationID != uuid.Nil {
			event.CorrelationID = metadata.CorrelationID.String()
		}
		if metadata.CausationID != uuid.Nil {
			event.CausationID = metadata.CausationID.String()
		}
	}
	return event
}

func toPosition(position *domain.Position) *Position {
	if position == nil {
		return nil
	}
	return &Position{TransactionID: position.TransactionID, Sequence: position.Sequence}
}

func parseListFilter(r *http.Request) (ListFilter, error) {
	filter := ListFilter{Limit: defaultLimit}

	if value := r.URL.Query().Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit <= 0 {
			return filter, errors.New("limit must be a positive number")
		}
		filter.Limit = min(limit, maxLimit)
	}

	after, err := parseAfter(r)
	if err != nil {
		return filter, err
	}
	filter.After = after

	filter.Types = r.URL.Query()["type"]
	filter.StreamPrefix = r.URL.Query().Get("streamPrefix")
	filter.CorrelationID = r.URL.Query().Get("correlationId")

	since, err := parseTime(r, "since")
	if err != nil {
		return filter, err
	}
	filter.Since = since

	until, err := parseTime(r, "until")
	if err != nil {
		return filter, err
	}
	filter.Until = until

	return filter, nil
}

func parseAfter(r *http.Request) (*domain.Position, error) {
	hasTxid := r.URL.Query().Has("afterTxid")
	hasSeq := r.URL.Query().Has("afterSeq")
	if !hasTxid && !hasSeq {
		return nil, nil
	}
	if hasTxid != hasSeq {
		return nil, errors.New("afterTxid and afterSeq must be given together")
	}

	txid, err := strconv.ParseUint(r.URL.Query().Get("afterTxid"), 10, 64)
	if err != nil {
		return nil, errors.New("afterTxid must be a non-negative number")
	}
	seq, err := strconv.ParseUint(r.URL.Query().Get("afterSeq"), 10, 64)
	if err != nil {
		return nil, errors.New("afterSeq must be a non-negative number")
	}
	return &domain.Position{TransactionID: txid, Sequence: seq}, nil
}

func parseTime(r *http.Request, name string) (*time.Time, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be an RFC3339 timestamp", name)
	}
	return &parsed, nil
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(response.Fail("sign in to use the event browser"))
}
