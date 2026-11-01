package item

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
	"github.com/hpcsc/stash/internal/use_cases/item/note"
)

var ErrAlreadyCaptured = errors.New("item is already captured")
var ErrAlreadyKept = errors.New("item is already kept")
var ErrAlreadyDropped = errors.New("item is already dropped")
var ErrNotDropped = errors.New("item is not in the trash")
var ErrNotCaptured = errors.New("item has not been captured")
var ErrMetadataAlreadyRecorded = errors.New("link metadata is already recorded")
var ErrThumbnailAlreadyRecorded = errors.New("thumbnail is already recorded")
var ErrFileTooLarge = errors.New("file is too large")
var ErrVideoRejected = errors.New("videos are not accepted")
var ErrNoteEmpty = errors.New("note must have text or an image")
var ErrTooManyImages = errors.New("note has too many images")
var ErrNoteTooLarge = errors.New("note is too large")
var ErrInvalidNoteBlock = errors.New("note block is invalid")
var ErrImageNotAttached = errors.New("image is not attached to the note")

var _ domain.Aggregate = (*Item)(nil)

func StreamID(user domain.UserID, id uuid.UUID) string {
	return domain.UserStream(user, "item-"+id.String())
}

type Item struct {
	domain.AggregateBase

	id              uuid.UUID
	state           string
	kind            string
	hadBeenKept     bool
	hasMetadata     bool
	hasThumbnail    bool
	tags            []uuid.UUID
	images          map[uuid.UUID]bool
	imageThumbnails map[uuid.UUID]bool
}

func (i *Item) ID() uuid.UUID {
	return i.id
}

func (i *Item) CaptureText(cmd command.CaptureText) error {
	if i.id != uuid.Nil {
		return ErrAlreadyCaptured
	}
	if cmd.Body == "" {
		return errors.New("body must not be empty")
	}

	i.record(domain.NewEvent(&event.TextCaptured{
		ItemID:     cmd.ItemID,
		Body:       cmd.Body,
		SourceApp:  cmd.SourceApp,
		SourceURL:  cmd.SourceURL,
		CapturedAt: cmd.CapturedAt,
	}))
	return nil
}

func (i *Item) CaptureLink(cmd command.CaptureLink) error {
	if i.id != uuid.Nil {
		return ErrAlreadyCaptured
	}
	if cmd.URL == "" {
		return errors.New("url must not be empty")
	}

	i.record(domain.NewEvent(&event.LinkCaptured{
		ItemID:     cmd.ItemID,
		URL:        cmd.URL,
		CapturedAt: cmd.CapturedAt,
	}))
	return nil
}

func (i *Item) CaptureImage(cmd command.CaptureImage) error {
	if i.id != uuid.Nil {
		return ErrAlreadyCaptured
	}
	if cmd.FileRef == "" {
		return errors.New("file ref must not be empty")
	}

	i.record(domain.NewEvent(&event.ImageCaptured{
		ItemID:     cmd.ItemID,
		FileRef:    cmd.FileRef,
		CapturedAt: cmd.CapturedAt,
	}))
	return nil
}

func (i *Item) CaptureFile(cmd command.CaptureFile) error {
	if i.id != uuid.Nil {
		return ErrAlreadyCaptured
	}
	if cmd.FileRef == "" {
		return errors.New("file ref must not be empty")
	}
	if cmd.FileSize > command.MaxFileSize {
		return ErrFileTooLarge
	}
	if strings.HasPrefix(cmd.MimeType, "video/") {
		return ErrVideoRejected
	}

	i.record(domain.NewEvent(&event.FileCaptured{
		ItemID:     cmd.ItemID,
		FileRef:    cmd.FileRef,
		FileName:   cmd.FileName,
		FileSize:   cmd.FileSize,
		MimeType:   cmd.MimeType,
		CapturedAt: cmd.CapturedAt,
	}))
	return nil
}

func (i *Item) CaptureNote(cmd command.CaptureNote) error {
	if i.id != uuid.Nil {
		return ErrAlreadyCaptured
	}
	if err := validateNote(cmd.Blocks); err != nil {
		return err
	}

	i.record(domain.NewEvent(&event.NoteCaptured{
		ItemID:     cmd.ItemID,
		Blocks:     cmd.Blocks,
		CapturedAt: cmd.CapturedAt,
	}))
	return nil
}

func (i *Item) RecordImageThumbnail(imageID uuid.UUID, thumbnailRef string) error {
	if i.id == uuid.Nil {
		return ErrNotCaptured
	}
	if _, ok := i.images[imageID]; !ok {
		return ErrImageNotAttached
	}
	if i.imageThumbnails[imageID] {
		return ErrThumbnailAlreadyRecorded
	}

	i.record(domain.NewEvent(&event.ImageThumbnailGenerated{
		ItemID:       i.id,
		ImageID:      imageID,
		ThumbnailRef: thumbnailRef,
	}))
	return nil
}

func (i *Item) RecordLinkMetadata(title, description string) error {
	if i.hasMetadata {
		return ErrMetadataAlreadyRecorded
	}

	i.record(domain.NewEvent(&event.LinkMetadataFetched{
		ItemID:      i.id,
		Title:       title,
		Description: description,
	}))
	return nil
}

func (i *Item) RecordThumbnail(thumbnailRef string) error {
	if i.hasThumbnail {
		return ErrThumbnailAlreadyRecorded
	}

	i.record(domain.NewEvent(&event.ThumbnailGenerated{
		ItemID:       i.id,
		ThumbnailRef: thumbnailRef,
	}))
	return nil
}

func (i *Item) Keep(cmd command.Keep) error {
	if i.id == uuid.Nil {
		return ErrNotCaptured
	}
	if i.state == "kept" {
		return ErrAlreadyKept
	}
	if i.state == "dropped" {
		return ErrAlreadyDropped
	}

	i.record(domain.NewEvent(&event.ItemKept{
		ItemID: cmd.ItemID,
	}))
	if len(cmd.TagIDs) > 0 {
		i.record(domain.NewEvent(&event.ItemTagged{
			ItemID: cmd.ItemID,
			TagIDs: cmd.TagIDs,
		}))
	}
	return nil
}

func (i *Item) Drop(cmd command.Drop) error {
	if i.id == uuid.Nil {
		return ErrNotCaptured
	}
	if i.state == "dropped" {
		return ErrAlreadyDropped
	}

	i.record(domain.NewEvent(&event.ItemDropped{
		ItemID: cmd.ItemID,
	}))
	return nil
}

func (i *Item) Restore(cmd command.Restore) error {
	if i.id == uuid.Nil {
		return ErrNotCaptured
	}
	if i.state != "dropped" {
		return ErrNotDropped
	}

	restoredTo := "inbox"
	if i.hadBeenKept {
		restoredTo = "library"
	}

	i.record(domain.NewEvent(&event.ItemRestored{
		ItemID:     cmd.ItemID,
		RestoredTo: restoredTo,
	}))
	return nil
}

func (i *Item) Delete(cmd command.Delete) error {
	if i.id == uuid.Nil {
		return ErrNotCaptured
	}
	if i.state != "dropped" {
		return ErrNotDropped
	}

	i.record(domain.NewEvent(&event.ItemDeleted{
		ItemID: cmd.ItemID,
	}))
	return nil
}

func (i *Item) Apply(e domain.Event) {
	switch p := e.Payload.(type) {
	case *event.TextCaptured:
		i.id = p.ItemID
		i.state = "inbox"
		i.kind = "text"
	case *event.LinkCaptured:
		i.id = p.ItemID
		i.state = "inbox"
		i.kind = "link"
	case *event.ImageCaptured:
		i.id = p.ItemID
		i.state = "inbox"
		i.kind = "image"
	case *event.FileCaptured:
		i.id = p.ItemID
		i.state = "inbox"
		i.kind = "file"
	case *event.NoteCaptured:
		i.id = p.ItemID
		i.state = "inbox"
		i.kind = "note"
		i.images = make(map[uuid.UUID]bool, len(p.Blocks))
		for _, b := range p.Blocks {
			if b.Image != nil {
				i.images[b.Image.ID] = true
			}
		}
		i.imageThumbnails = make(map[uuid.UUID]bool)
	case *event.LinkMetadataFetched:
		i.hasMetadata = true
	case *event.ThumbnailGenerated:
		i.hasThumbnail = true
	case *event.ImageThumbnailGenerated:
		if i.imageThumbnails == nil {
			i.imageThumbnails = make(map[uuid.UUID]bool)
		}
		i.imageThumbnails[p.ImageID] = true
	case *event.ItemKept:
		i.state = "kept"
		i.hadBeenKept = true
	case *event.ItemDropped:
		i.state = "dropped"
	case *event.ItemRestored:
		i.state = "inbox"
		if p.RestoredTo == "library" {
			i.state = "kept"
		}
	case *event.ItemDeleted:
		i.state = "deleted"
	case *event.ItemTagged:
		i.tags = p.TagIDs
	}
}

func (i *Item) record(e domain.Event) {
	i.Apply(e)
	i.Record(e)
}

func validateNote(blocks []note.Block) error {
	if len(blocks) == 0 {
		return ErrNoteEmpty
	}

	hasText := false
	imageCount := 0
	var totalSize int64
	seen := make(map[uuid.UUID]bool, len(blocks))

	for _, b := range blocks {
		switch b.Type {
		case note.TypeText:
			if strings.TrimSpace(b.Text) != "" {
				hasText = true
			}
		case note.TypeImage:
			if b.Image == nil || b.Image.ID == uuid.Nil || b.Image.FileRef == "" {
				return ErrInvalidNoteBlock
			}
			if seen[b.Image.ID] {
				return ErrInvalidNoteBlock
			}
			seen[b.Image.ID] = true
			if strings.HasPrefix(b.Image.MimeType, "video/") {
				return ErrVideoRejected
			}
			if !strings.HasPrefix(b.Image.MimeType, "image/") {
				return ErrInvalidNoteBlock
			}
			if b.Image.FileSize > command.MaxFileSize {
				return ErrFileTooLarge
			}
			imageCount++
			totalSize += b.Image.FileSize
		default:
			return ErrInvalidNoteBlock
		}
	}

	if !hasText && imageCount == 0 {
		return ErrNoteEmpty
	}
	if imageCount > command.MaxNoteImages {
		return ErrTooManyImages
	}
	if totalSize > command.MaxNoteSize {
		return ErrNoteTooLarge
	}
	return nil
}
