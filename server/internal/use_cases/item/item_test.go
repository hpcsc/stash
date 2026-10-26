//go:build unit

package item_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hpcsc/stash/internal/domain"
	"github.com/hpcsc/stash/internal/use_cases/item"
	"github.com/hpcsc/stash/internal/use_cases/item/command"
	"github.com/hpcsc/stash/internal/use_cases/item/event"
	"github.com/stretchr/testify/require"
)

func TestItem(t *testing.T) {
	t.Run("capture text", func(t *testing.T) {
		t.Run("captures text into the inbox", func(t *testing.T) {
			var i item.Item
			cmd := captureText()

			require.NoError(t, i.CaptureText(cmd))

			require.Equal(t, &event.TextCaptured{ItemID: cmd.ItemID, Body: "hello", SourceApp: "chrome", SourceURL: "https://example.com", CapturedAt: capturedAt}, onlyRecorded(t, &i).Payload)
			require.Equal(t, cmd.ItemID, i.ID())
		})

		t.Run("rejects a second capture", func(t *testing.T) {
			i := given(captured())

			require.ErrorIs(t, i.CaptureText(captureText()), item.ErrAlreadyCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects an empty body", func(t *testing.T) {
			var i item.Item
			cmd := captureText()
			cmd.Body = ""

			require.Error(t, i.CaptureText(cmd))
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("capture link", func(t *testing.T) {
		t.Run("captures a link into the inbox", func(t *testing.T) {
			var i item.Item
			cmd := captureLink()

			require.NoError(t, i.CaptureLink(cmd))

			require.Equal(t, &event.LinkCaptured{
				ItemID:     cmd.ItemID,
				URL:        "https://example.com",
				CapturedAt: capturedAt,
			}, onlyRecorded(t, &i).Payload)
			require.Equal(t, cmd.ItemID, i.ID())
		})

		t.Run("rejects a second capture", func(t *testing.T) {
			i := given(linkCaptured())

			require.ErrorIs(t, i.CaptureLink(captureLink()), item.ErrAlreadyCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects an empty URL", func(t *testing.T) {
			var i item.Item
			cmd := captureLink()
			cmd.URL = ""

			require.Error(t, i.CaptureLink(cmd))
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("record link metadata", func(t *testing.T) {
		t.Run("records title and description for a link item", func(t *testing.T) {
			i := given(linkCaptured())

			require.NoError(t, i.RecordLinkMetadata("Example Page", "A page for examples"))

			require.Equal(t, &event.LinkMetadataFetched{
				ItemID:      itemID,
				Title:       "Example Page",
				Description: "A page for examples",
			}, onlyRecorded(t, i).Payload)
		})

		t.Run("records metadata with only title", func(t *testing.T) {
			i := given(linkCaptured())

			require.NoError(t, i.RecordLinkMetadata("Example Page", ""))

			require.Equal(t, &event.LinkMetadataFetched{
				ItemID: itemID,
				Title:  "Example Page",
			}, onlyRecorded(t, i).Payload)
		})

		t.Run("rejects metadata for an already-enriched link", func(t *testing.T) {
			i := given(linkCaptured(), linkMetadataFetched())

			require.ErrorIs(t, i.RecordLinkMetadata("New Title", ""), item.ErrMetadataAlreadyRecorded)
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("capture image", func(t *testing.T) {
		t.Run("captures an image into the inbox", func(t *testing.T) {
			var i item.Item
			cmd := captureImage()

			require.NoError(t, i.CaptureImage(cmd))

			require.Equal(t, &event.ImageCaptured{
				ItemID:     cmd.ItemID,
				FileRef:    "images/photo.jpg",
				CapturedAt: capturedAt,
			}, onlyRecorded(t, &i).Payload)
			require.Equal(t, cmd.ItemID, i.ID())
		})

		t.Run("rejects a second capture", func(t *testing.T) {
			i := given(imageCaptured())

			require.ErrorIs(t, i.CaptureImage(captureImage()), item.ErrAlreadyCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects an empty file ref", func(t *testing.T) {
			var i item.Item
			cmd := captureImage()
			cmd.FileRef = ""

			require.Error(t, i.CaptureImage(cmd))
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("capture file", func(t *testing.T) {
		t.Run("captures a file into the inbox", func(t *testing.T) {
			var i item.Item
			cmd := captureFile()

			require.NoError(t, i.CaptureFile(cmd))

			require.Equal(t, &event.FileCaptured{
				ItemID:     cmd.ItemID,
				FileRef:    "files/report.pdf",
				FileName:   "report.pdf",
				FileSize:   1024,
				MimeType:   "application/pdf",
				CapturedAt: capturedAt,
			}, onlyRecorded(t, &i).Payload)
			require.Equal(t, cmd.ItemID, i.ID())
		})

		t.Run("rejects a second capture", func(t *testing.T) {
			i := given(fileCaptured())

			require.ErrorIs(t, i.CaptureFile(captureFile()), item.ErrAlreadyCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects an empty file ref", func(t *testing.T) {
			var i item.Item
			cmd := captureFile()
			cmd.FileRef = ""

			require.Error(t, i.CaptureFile(cmd))
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects a file larger than the limit", func(t *testing.T) {
			var i item.Item
			cmd := captureFile()
			cmd.FileSize = command.MaxFileSize + 1

			require.ErrorIs(t, i.CaptureFile(cmd), item.ErrFileTooLarge)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects a video", func(t *testing.T) {
			var i item.Item
			cmd := captureFile()
			cmd.MimeType = "video/mp4"

			require.ErrorIs(t, i.CaptureFile(cmd), item.ErrVideoRejected)
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("record thumbnail", func(t *testing.T) {
		t.Run("records a thumbnail for an image item", func(t *testing.T) {
			i := given(imageCaptured())

			require.NoError(t, i.RecordThumbnail("thumbnails/photo.jpg"))

			require.Equal(t, &event.ThumbnailGenerated{
				ItemID:       itemID,
				ThumbnailRef: "thumbnails/photo.jpg",
			}, onlyRecorded(t, i).Payload)
		})

		t.Run("rejects thumbnail for an image that already has one", func(t *testing.T) {
			i := given(imageCaptured(), thumbnailGenerated())

			require.ErrorIs(t, i.RecordThumbnail("new-thumb.jpg"), item.ErrThumbnailAlreadyRecorded)
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("keep", func(t *testing.T) {
		t.Run("keeps an inbox item", func(t *testing.T) {
			i := given(captured())

			require.NoError(t, i.Keep(keepCmd()))

			require.Equal(t, &event.ItemKept{ItemID: itemID}, onlyRecorded(t, i).Payload)
		})

		t.Run("keeps an inbox item with tags", func(t *testing.T) {
			i := given(captured())
			cmd := keepCmd()
			cmd.TagIDs = []uuid.UUID{uuid.New(), uuid.New()}

			require.NoError(t, i.Keep(cmd))

			recorded := i.Recorded()
			require.Len(t, recorded, 2)
			require.Equal(t, &event.ItemKept{ItemID: itemID}, recorded[0].Payload)
			require.Equal(t, cmd.TagIDs, recorded[1].Payload.(*event.ItemTagged).TagIDs)
		})

		t.Run("rejects keeping an item that is not captured", func(t *testing.T) {
			var i item.Item

			require.ErrorIs(t, i.Keep(keepCmd()), item.ErrNotCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects keeping an already-kept item", func(t *testing.T) {
			i := given(captured(), kept())

			require.ErrorIs(t, i.Keep(keepCmd()), item.ErrAlreadyKept)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects keeping a dropped item", func(t *testing.T) {
			i := given(captured(), dropped())

			require.ErrorIs(t, i.Keep(keepCmd()), item.ErrAlreadyDropped)
			require.Empty(t, i.Recorded())
		})
	})

	t.Run("drop", func(t *testing.T) {
		t.Run("drops an inbox item", func(t *testing.T) {
			i := given(captured())

			require.NoError(t, i.Drop(dropCmd()))

			require.Equal(t, &event.ItemDropped{ItemID: itemID}, onlyRecorded(t, i).Payload)
		})

		t.Run("rejects dropping an item that is not captured", func(t *testing.T) {
			var i item.Item

			require.ErrorIs(t, i.Drop(dropCmd()), item.ErrNotCaptured)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects dropping an already-dropped item", func(t *testing.T) {
			i := given(captured(), dropped())

			require.ErrorIs(t, i.Drop(dropCmd()), item.ErrAlreadyDropped)
			require.Empty(t, i.Recorded())
		})

		t.Run("rejects dropping a kept item", func(t *testing.T) {
			i := given(captured(), kept())

			require.ErrorIs(t, i.Drop(dropCmd()), item.ErrAlreadyKept)
			require.Empty(t, i.Recorded())
		})
	})
}

var (
	itemID     = uuid.MustParse("6f1c2a54-7b0e-4f6a-9a51-2d8f0c3e9b71")
	capturedAt = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
)

func captureText() command.CaptureText {
	return command.CaptureText{ItemID: itemID, Body: "hello", SourceApp: "chrome", SourceURL: "https://example.com", CapturedAt: capturedAt}
}

func captureLink() command.CaptureLink {
	return command.CaptureLink{
		ItemID:     itemID,
		URL:        "https://example.com",
		CapturedAt: capturedAt,
	}
}

func captureImage() command.CaptureImage {
	return command.CaptureImage{
		ItemID:     itemID,
		FileRef:    "images/photo.jpg",
		CapturedAt: capturedAt,
	}
}

func captureFile() command.CaptureFile {
	return command.CaptureFile{
		ItemID:     itemID,
		FileRef:    "files/report.pdf",
		FileName:   "report.pdf",
		FileSize:   1024,
		MimeType:   "application/pdf",
		CapturedAt: capturedAt,
	}
}

func keepCmd() command.Keep {
	return command.Keep{ItemID: itemID}
}

func dropCmd() command.Drop {
	return command.Drop{ItemID: itemID}
}

func given(history ...domain.Event) *item.Item {
	var i item.Item
	for _, e := range history {
		i.Apply(e)
	}
	i.MarkCommitted(uint64(len(history)))
	return &i
}

func captured() domain.Event {
	return domain.NewEvent(&event.TextCaptured{ItemID: itemID, Body: "hello", CapturedAt: capturedAt})
}

func linkCaptured() domain.Event {
	return domain.NewEvent(&event.LinkCaptured{ItemID: itemID, URL: "https://example.com", CapturedAt: capturedAt})
}

func linkMetadataFetched() domain.Event {
	return domain.NewEvent(&event.LinkMetadataFetched{ItemID: itemID, Title: "Example Page", Description: "A page for examples"})
}

func imageCaptured() domain.Event {
	return domain.NewEvent(&event.ImageCaptured{ItemID: itemID, FileRef: "images/photo.jpg", CapturedAt: capturedAt})
}

func fileCaptured() domain.Event {
	return domain.NewEvent(&event.FileCaptured{
		ItemID:     itemID,
		FileRef:    "files/report.pdf",
		FileName:   "report.pdf",
		FileSize:   1024,
		MimeType:   "application/pdf",
		CapturedAt: capturedAt,
	})
}

func thumbnailGenerated() domain.Event {
	return domain.NewEvent(&event.ThumbnailGenerated{ItemID: itemID, ThumbnailRef: "thumbnails/photo.jpg"})
}

func kept() domain.Event {
	return domain.NewEvent(&event.ItemKept{ItemID: itemID})
}

func dropped() domain.Event {
	return domain.NewEvent(&event.ItemDropped{ItemID: itemID})
}

func onlyRecorded(t *testing.T, i *item.Item) domain.Event {
	recorded := i.Recorded()
	require.Len(t, recorded, 1)
	return recorded[0]
}
