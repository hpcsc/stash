package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"log/slog"
	"strings"

	"golang.org/x/image/draw"

	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/domain/reaction"
	"github.com/hpcsc/stash/internal/domain/reaction/dispatch"
	"github.com/hpcsc/stash/internal/storage"
	"github.com/hpcsc/stash/internal/use_cases/item"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
)

const maxWidth = 200

type Reaction struct {
	items  *item.Repository
	store  storage.Storage
	logger *slog.Logger
}

var _ reaction.Interface = (*Reaction)(nil)

func New(items *item.Repository, store storage.Storage, logger *slog.Logger) *Reaction {
	return &Reaction{
		items:  items,
		store:  store,
		logger: logger,
	}
}

func (r *Reaction) Name() string {
	return "thumbnail"
}

func (r *Reaction) EventTypes() []string {
	return []string{
		event.ImageCaptured{}.EventType(),
		event.NoteCaptured{}.EventType(),
	}
}

func (r *Reaction) Handle(ctx context.Context, evt *domain.Event) error {
	user, err := domain.UserOf(evt.StreamID)
	if err != nil {
		return err
	}

	switch captured := evt.Payload.(type) {
	case *event.ImageCaptured:
		return r.handleImage(ctx, evt, user, captured)
	case *event.NoteCaptured:
		return r.handleNote(ctx, evt, user, captured)
	}
	return nil
}

func (r *Reaction) handleImage(ctx context.Context, evt *domain.Event, user domain.UserID, captured *event.ImageCaptured) error {
	thumbnailRef, err := r.generate(ctx, captured.FileRef)
	if err != nil {
		r.logger.Warn("failed to generate thumbnail",
			"fileRef", captured.FileRef,
			"itemId", captured.ItemID,
			"error", err,
		)
		return nil
	}

	items := r.items.For(user)
	i, err := items.Load(ctx, captured.ItemID)
	if err != nil {
		return fmt.Errorf("loading item %s for thumbnail: %w", captured.ItemID, err)
	}

	if err := i.RecordThumbnail(thumbnailRef); err != nil {
		if err == item.ErrThumbnailAlreadyRecorded {
			return nil
		}
		return err
	}

	return items.Save(ctx, i, dispatch.CommandIDFor(r.Name(), evt))
}

func (r *Reaction) handleNote(ctx context.Context, evt *domain.Event, user domain.UserID, captured *event.NoteCaptured) error {
	items := r.items.For(user)
	i, err := items.Load(ctx, captured.ItemID)
	if err != nil {
		return fmt.Errorf("loading item %s for thumbnails: %w", captured.ItemID, err)
	}

	recorded := false
	for _, b := range captured.Blocks {
		if b.Image == nil {
			continue
		}

		thumbnailRef, err := r.generate(ctx, b.Image.FileRef)
		if err != nil {
			r.logger.Warn("failed to generate thumbnail",
				"fileRef", b.Image.FileRef,
				"itemId", captured.ItemID,
				"imageId", b.Image.ID,
				"error", err,
			)
			continue
		}

		if err := i.RecordImageThumbnail(b.Image.ID, thumbnailRef); err != nil {
			if err == item.ErrThumbnailAlreadyRecorded {
				continue
			}
			r.logger.Warn("failed to record image thumbnail",
				"itemId", captured.ItemID,
				"imageId", b.Image.ID,
				"error", err,
			)
			continue
		}
		recorded = true
	}

	if !recorded {
		return nil
	}
	return items.Save(ctx, i, dispatch.CommandIDFor(r.Name(), evt))
}

func (r *Reaction) generate(ctx context.Context, fileRef string) (string, error) {
	reader, err := r.store.Open(ctx, fileRef)
	if err != nil {
		return "", fmt.Errorf("opening image %s: %w", fileRef, err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return "", fmt.Errorf("decoding image %s: %w", fileRef, err)
	}

	thumbnail := resize(img, maxWidth)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 80}); err != nil {
		return "", fmt.Errorf("encoding thumbnail: %w", err)
	}

	thumbnailRef := thumbnailKey(fileRef)
	if _, err := r.store.Save(ctx, thumbnailRef, &buf); err != nil {
		return "", fmt.Errorf("saving thumbnail: %w", err)
	}

	return thumbnailRef, nil
}

func resize(src image.Image, width int) image.Image {
	bounds := src.Bounds()
	if bounds.Dx() <= width {
		return src
	}

	ratio := float64(width) / float64(bounds.Dx())
	height := int(float64(bounds.Dy()) * ratio)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

func thumbnailKey(fileRef string) string {
	name := strings.TrimPrefix(fileRef, "images/")
	return "thumbnails/" + name
}
