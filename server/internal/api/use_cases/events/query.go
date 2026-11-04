package events

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultLimit     = 100
	maxLimit         = 500
	tailBatchSize    = 500
	tailPollInterval = time.Second
)

var ErrUnknownProjection = errors.New("unknown projection")

const eventColumns = `id::text, stream_id, version, transaction_id, sequence, type, schema_version, data, metadata, created_at`

type StoredEvent struct {
	ID            string
	StreamID      string
	Version       uint64
	Position      domain.Position
	Type          string
	SchemaVersion int
	Data          []byte
	Metadata      []byte
	CreatedAt     time.Time
}

type ListFilter struct {
	After         *domain.Position
	Limit         int
	Types         []string
	StreamPrefix  string
	CorrelationID string
	Since         *time.Time
	Until         *time.Time
}

type CheckpointRow struct {
	Name      string
	Position  domain.Position
	UpdatedAt time.Time
}

func NewQuery(pool *pgxpool.Pool) *Query {
	return &Query{pool: pool}
}

type Query struct {
	pool *pgxpool.Pool
}

func (q *Query) Scoped(scope Scope) Reader {
	return &reader{pool: q.pool, scope: scope}
}

type Reader interface {
	List(ctx context.Context, filter ListFilter) ([]StoredEvent, *domain.Position, error)
	Stream(ctx context.Context, streamID string) ([]StoredEvent, error)
	Streams(ctx context.Context) ([]StreamSummary, error)
	Types(ctx context.Context) ([]string, error)
	Checkpoints(ctx context.Context) ([]CheckpointRow, *domain.Position, error)
	CountAfter(ctx context.Context, position domain.Position, eventTypes []string) (uint64, error)
	NextEvent(ctx context.Context, position domain.Position, eventTypes []string) (*NextEvent, error)
	Projection(ctx context.Context, name string) (*ProjectionTable, error)
	Tail(ctx context.Context, after *domain.Position, emit func(StoredEvent) error) error
}

type reader struct {
	pool  *pgxpool.Pool
	scope Scope
}

func (r *reader) List(ctx context.Context, filter ListFilter) ([]StoredEvent, *domain.Position, error) {
	if filter.Limit <= 0 {
		filter.Limit = defaultLimit
	}
	if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}

	clauses := []string{"TRUE"}
	args := []any{}
	argIdx := 1

	if filter.After != nil {
		clauses = append(clauses, fmt.Sprintf("(transaction_id, sequence) > ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, filter.After.TransactionID, filter.After.Sequence)
		argIdx += 2
	}
	if len(filter.Types) > 0 {
		clauses = append(clauses, fmt.Sprintf("type = ANY($%d)", argIdx))
		args = append(args, filter.Types)
		argIdx++
	}
	if filter.StreamPrefix != "" {
		clauses = append(clauses, fmt.Sprintf("stream_id LIKE $%d || '%%'", argIdx))
		args = append(args, filter.StreamPrefix)
		argIdx++
	}
	if filter.CorrelationID != "" {
		clauses = append(clauses, fmt.Sprintf("metadata->>'CorrelationID' = $%d", argIdx))
		args = append(args, filter.CorrelationID)
		argIdx++
	}
	if filter.Since != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.Since)
		argIdx++
	}
	if filter.Until != nil {
		clauses = append(clauses, fmt.Sprintf("created_at < $%d", argIdx))
		args = append(args, *filter.Until)
		argIdx++
	}
	clauses, args = r.scopeClause(clauses, args, argIdx)

	query := `SELECT ` + eventColumns + ` FROM events WHERE ` + strings.Join(clauses, " AND ")
	query += fmt.Sprintf(" ORDER BY transaction_id, sequence LIMIT %d", filter.Limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	events, err := collectStoredEvents(rows)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list events: %w", err)
	}

	var next *domain.Position
	if len(events) > filter.Limit {
		events = events[:filter.Limit]
		last := events[len(events)-1]
		next = &domain.Position{TransactionID: last.Position.TransactionID, Sequence: last.Position.Sequence}
	}
	return events, next, nil
}

func (r *reader) Stream(ctx context.Context, streamID string) ([]StoredEvent, error) {
	clauses := []string{"stream_id = $1"}
	args := []any{streamID}
	clauses, args = r.scopeClause(clauses, args, 2)

	query := `SELECT ` + eventColumns + ` FROM events WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY version`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream %s: %w", streamID, err)
	}
	defer rows.Close()

	events, err := collectStoredEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream %s: %w", streamID, err)
	}
	return events, nil
}

func (r *reader) Streams(ctx context.Context) ([]StreamSummary, error) {
	clauses, args := r.scopeClause([]string{"TRUE"}, []any{}, 1)

	query := `SELECT stream_id, COUNT(*), MAX(version), MAX(created_at) FROM events WHERE ` + strings.Join(clauses, " AND ") + ` GROUP BY stream_id ORDER BY stream_id`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list streams: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StreamSummary, error) {
		var s StreamSummary
		err := row.Scan(&s.StreamID, &s.EventCount, &s.LastVersion, &s.LastCreatedAt)
		return s, err
	})
}

func (r *reader) Types(ctx context.Context) ([]string, error) {
	clauses, args := r.scopeClause([]string{}, []any{}, 1)

	query := `SELECT DISTINCT type FROM events`
	if len(clauses) > 0 {
		query += ` WHERE ` + strings.Join(clauses, " AND ")
	}
	query += ` ORDER BY type`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list event types: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var t string
		err := row.Scan(&t)
		return t, err
	})
}

func (r *reader) Checkpoints(ctx context.Context) ([]CheckpointRow, *domain.Position, error) {
	rows, err := r.pool.Query(ctx, `SELECT name, transaction_id, sequence, updated_at FROM checkpoints ORDER BY name`)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	checkpoints, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (CheckpointRow, error) {
		var c CheckpointRow
		err := row.Scan(&c.Name, &c.Position.TransactionID, &c.Position.Sequence, &c.UpdatedAt)
		return c, err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	head, err := r.lastPosition(ctx)
	if err != nil {
		return nil, nil, err
	}
	return checkpoints, head, nil
}

func (r *reader) CountAfter(ctx context.Context, position domain.Position, eventTypes []string) (uint64, error) {
	var count uint64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM events WHERE sequence > $1 AND type = ANY($2)`,
		position.Sequence, eventTypes,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events after %d: %w", position.Sequence, err)
	}
	return count, nil
}

func (r *reader) NextEvent(ctx context.Context, position domain.Position, eventTypes []string) (*NextEvent, error) {
	var next NextEvent
	err := r.pool.QueryRow(ctx,
		`SELECT transaction_id, sequence, type, stream_id FROM events WHERE sequence > $1 AND type = ANY($2) ORDER BY sequence LIMIT 1`,
		position.Sequence, eventTypes,
	).Scan(&next.Position.TransactionID, &next.Position.Sequence, &next.Type, &next.StreamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the next event after %d: %w", position.Sequence, err)
	}
	return &next, nil
}

func (r *reader) Projection(ctx context.Context, name string) (*ProjectionTable, error) {
	table, columns, where, order, err := projectionColumns(name)
	if err != nil {
		return nil, err
	}

	selects := make([]string, len(columns))
	for i, column := range columns {
		selects[i] = column + "::text"
	}

	query := "SELECT " + strings.Join(selects, ", ") + " FROM " + table
	args := []any{}
	if !r.scope.IsAll() {
		query += " WHERE " + where
		args = append(args, r.scope.User().String())
	}
	query += " ORDER BY " + order

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read the %s projection: %w", name, err)
	}
	defer rows.Close()

	result := &ProjectionTable{Columns: columns, Rows: [][]string{}}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to read a %s row: %w", name, err)
		}
		row := make([]string, len(values))
		for i, value := range values {
			if value != nil {
				row[i] = fmt.Sprint(value)
			}
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func projectionColumns(name string) (table string, columns []string, where, order string, err error) {
	switch name {
	case "items":
		return "items",
			[]string{"id", "user_id", "type", "body", "state", "title", "description", "file_path", "thumbnail_path", "file_name", "file_size", "mime_type", "captured_at", "version", "created_at", "updated_at"},
			"user_id = $1",
			"captured_at DESC",
			nil
	case "tags":
		return "tags",
			[]string{"id", "user_id", "name", "version", "created_at", "updated_at"},
			"user_id = $1",
			"LOWER(name)",
			nil
	case "item-tags":
		return "item_tags",
			[]string{"item_id", "tag_id"},
			"item_id IN (SELECT id FROM items WHERE user_id = $1)",
			"item_id, tag_id",
			nil
	default:
		return "", nil, "", "", ErrUnknownProjection
	}
}

func (r *reader) Tail(ctx context.Context, after *domain.Position, emit func(StoredEvent) error) error {
	position := &domain.Position{}
	if after != nil {
		*position = *after
	} else {
		head, err := r.lastPosition(ctx)
		if err != nil {
			return err
		}
		if head != nil {
			*position = *head
		}
	}

	for {
		events, err := r.readAfter(ctx, position)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !wait(ctx, tailPollInterval) {
				return ctx.Err()
			}
			continue
		}

		for _, event := range events {
			if err := emit(event); err != nil {
				return err
			}
			position.TransactionID = event.Position.TransactionID
			position.Sequence = event.Position.Sequence
		}

		if !wait(ctx, tailPollInterval) {
			return ctx.Err()
		}
	}
}

func (r *reader) readAfter(ctx context.Context, after *domain.Position) ([]StoredEvent, error) {
	clauses := []string{
		"(transaction_id, sequence) > ($1, $2)",
		"transaction_id < pg_snapshot_xmin(pg_current_snapshot())",
	}
	args := []any{after.TransactionID, after.Sequence}
	clauses, args = r.scopeClause(clauses, args, 3)

	query := `SELECT ` + eventColumns + ` FROM events WHERE ` + strings.Join(clauses, " AND ")
	query += fmt.Sprintf(" ORDER BY transaction_id, sequence LIMIT %d", tailBatchSize)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	defer rows.Close()

	return collectStoredEvents(rows)
}

func (r *reader) lastPosition(ctx context.Context) (*domain.Position, error) {
	var position domain.Position
	err := r.pool.QueryRow(ctx,
		`SELECT transaction_id, sequence FROM events WHERE transaction_id < pg_snapshot_xmin(pg_current_snapshot()) ORDER BY transaction_id DESC, sequence DESC LIMIT 1`,
	).Scan(&position.TransactionID, &position.Sequence)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the last position: %w", err)
	}
	return &position, nil
}

func (r *reader) scopeClause(clauses []string, args []any, argIdx int) ([]string, []any) {
	if r.scope.IsAll() {
		return clauses, args
	}
	clauses = append(clauses, fmt.Sprintf("(stream_id LIKE $%d || '/%%' OR stream_id = $%d)", argIdx, argIdx+1))
	args = append(args, "user/"+r.scope.User().String(), "identity-"+r.scope.User().String())
	return clauses, args
}

func collectStoredEvents(rows pgx.Rows) ([]StoredEvent, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredEvent, error) {
		var e StoredEvent
		err := row.Scan(
			&e.ID, &e.StreamID, &e.Version,
			&e.Position.TransactionID, &e.Position.Sequence,
			&e.Type, &e.SchemaVersion, &e.Data, &e.Metadata, &e.CreatedAt,
		)
		return e, err
	})
}

func wait(ctx context.Context, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
