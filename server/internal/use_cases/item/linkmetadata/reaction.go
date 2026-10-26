package linkmetadata

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/reaction"
	"github.com/hpcsc/stash/internal/domain/reaction/dispatch"
	"github.com/hpcsc/stash/internal/use_cases/item"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
)

var titleRe = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
var descRe = regexp.MustCompile(`(?i)<meta[^>]+name=["']description["'][^>]+content=["']([^"']+)["']`)

type linkMeta struct {
	title       string
	description string
}

func fetchTitle(url string) (linkMeta, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return linkMeta{}, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return linkMeta{}, fmt.Errorf("non-200 response from %s: %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return linkMeta{}, err
	}

	html := string(body)
	var meta linkMeta

	if m := titleRe.FindStringSubmatch(html); len(m) > 1 {
		meta.title = strings.TrimSpace(m[1])
	}
	if m := descRe.FindStringSubmatch(html); len(m) > 1 {
		meta.description = strings.TrimSpace(m[1])
	}

	return meta, nil
}

type Reaction struct {
	items  *item.Repository
	logger *slog.Logger
}

var _ reaction.Interface = (*Reaction)(nil)

func New(items *item.Repository, logger *slog.Logger) *Reaction {
	return &Reaction{
		items:  items,
		logger: logger,
	}
}

func (r *Reaction) Name() string {
	return "link-metadata"
}

func (r *Reaction) EventTypes() []string {
	return []string{event.LinkCaptured{}.EventType()}
}

func (r *Reaction) Handle(ctx context.Context, evt *domain.Event) error {
	captured, ok := evt.Payload.(*event.LinkCaptured)
	if !ok {
		return nil
	}

	meta, err := fetchTitle(captured.URL)
	if err != nil {
		r.logger.Warn("failed to fetch link metadata",
			"url", captured.URL,
			"itemId", captured.ItemID,
			"error", err,
		)
		return nil
	}

	if meta.title == "" && meta.description == "" {
		return nil
	}

	i, err := r.items.Load(ctx, captured.ItemID)
	if err != nil {
		return fmt.Errorf("loading item %s for metadata enrichment: %w", captured.ItemID, err)
	}

	if err := i.RecordLinkMetadata(meta.title, meta.description); err != nil {
		if err == item.ErrMetadataAlreadyRecorded {
			return nil
		}
		return err
	}

	return r.items.Save(ctx, i, dispatch.CommandIDFor(r.Name(), evt))
}