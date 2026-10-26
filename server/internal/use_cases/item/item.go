package item

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
)

var ErrAlreadyCaptured = errors.New("item is already captured")
var ErrAlreadyKept = errors.New("item is already kept")
var ErrAlreadyDropped = errors.New("item is already dropped")
var ErrNotCaptured = errors.New("item has not been captured")
var ErrMetadataAlreadyRecorded = errors.New("link metadata is already recorded")
var ErrThumbnailAlreadyRecorded = errors.New("thumbnail is already recorded")

var _ domain.Aggregate = (*Item)(nil)

func StreamID(id uuid.UUID) string {
	return fmt.Sprintf("item-%s", id)
}

type Item struct {
	domain.AggregateBase

	id           uuid.UUID
	state        string
	kind         string
	hasMetadata  bool
	hasThumbnail bool
	tags         []uuid.UUID
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
	if i.state == "kept" {
		return ErrAlreadyKept
	}

	i.record(domain.NewEvent(&event.ItemDropped{
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
	case *event.LinkMetadataFetched:
		i.hasMetadata = true
	case *event.ThumbnailGenerated:
		i.hasThumbnail = true
	case *event.ItemKept:
		i.state = "kept"
	case *event.ItemDropped:
		i.state = "dropped"
	case *event.ItemTagged:
		i.tags = p.TagIDs
	}
}

func (i *Item) record(e domain.Event) {
	i.Apply(e)
	i.Record(e)
}
