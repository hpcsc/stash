# Capture Image + Thumbnail

**Story:** As a user, I want to save an image and see a thumbnail in my Inbox.

**Acceptance Criteria:**
- Upload an image via multipart `POST /items` → appears in the Inbox as type "image"
- A thumbnail is generated asynchronously by a reactor
- The Inbox card shows the thumbnail (or a placeholder while generating)
- The item detail view shows the full image

**Depends on:** Slice 3 (Link capture + metadata) — for the reactor pattern and WebSocket push

---

## 1. Storage interface and disk implementation

Filesystem abstraction for saving/retrieving files, needed before the image capture can store anything.

**Files:**
- `internal/storage/storage.go` — `type Storage interface { Save(ctx, key string, reader io.Reader) (string, error); Open(ctx, key string) (io.ReadCloser, error) }`
- `internal/storage/disk.go` — disk-backed implementation, writes to a configurable directory
- `internal/common/config/config.go` — add `StorageDir` and `StorageBaseURL` fields

No tests yet — the interface itself is trivial; disk implementation gets tested with the integration tests.

---

## 2. Migration: add file columns to items table

**Files:**
- `migrations/<timestamp>_add-file-columns-to-items.up.sql` — `ALTER TABLE items ADD COLUMN IF NOT EXISTS file_path TEXT NOT NULL DEFAULT ''`
- `migrations/<timestamp>_add-file-columns-to-items.down.sql` — drop the columns

---

## 3. Command and event types

**Files:**
- `use_cases/item/command/capture_image.go` — `CaptureImage` struct with `ItemID`, `FileRef`, `CapturedAt`
- `use_cases/item/event/image_captured.go` — `ImageCaptured` struct with `ItemID`, `FileRef`, `CapturedAt`, `EventType()` returning `"stash.image-captured"`
- `use_cases/item/event/thumbnail_generated.go` — `ThumbnailGenerated` struct with `ItemID`, `ThumbnailRef`, `EventType()` returning `"stash.thumbnail-generated"`

---

## 4. Aggregate: CaptureImage + RecordThumbnail methods

**Files:**
- `use_cases/item/item.go` — add:
  - `CaptureImage(cmd command.CaptureImage) error` — validates `FileRef` non-empty, records `ImageCaptured`, refused when already captured
  - `RecordThumbnail(thumbnailRef string) error` — records `ThumbnailGenerated`, refused when `hasThumbnail` already true
  - `hasThumbnail bool` field on `Item` struct
  - `Apply()` cases for `ImageCaptured` (sets id, state, kind="image") and `ThumbnailGenerated` (sets hasThumbnail=true)
  - `ErrThumbnailAlreadyRecorded` error

**Tests:**
- `use_cases/item/item_test.go` — add "capture image" and "record thumbnail" test groups (same pattern as capture link + record metadata)

---

## 5. Event registration

**Files:**
- `use_cases/item/events.go` — register `ImageCaptured` and `ThumbnailGenerated`

---

## 6. Use case handler

**Files:**
- `use_cases/item/capture_image/handler.go` — `Register(bus, items)` + handler that loads aggregate, calls `CaptureImage()`, saves. Same pattern as `capture_link/handler.go`.
- `use_cases/register.go` — call `capture_image.Register(bus, items)` in `RegisterHandlers`

---

## 7. Projection: handle ImageCaptured and ThumbnailGenerated

**Files:**
- `use_cases/item/items/projection.go` — add `ImageCaptured` and `ThumbnailGenerated` to `EventTypes()`, add cases:
  - `ImageCaptured` → `INSERT INTO items (id, type, body, file_path, captured_at, state, version) VALUES ($1, 'image', '', $2, $3, 'inbox', $4)`
  - `ThumbnailGenerated` → `UPDATE items SET thumbnail_path = $2, version = $3, updated_at = NOW() WHERE id = $1`

**Tests:**
- `use_cases/item/items/projection_test.go` — add test for projecting an `ImageCaptured` event (integration test, needs DB)

---

## 8. Read model: add file_path and thumbnail_path to queries

**Files:**
- `use_cases/item/items/query.go` — add `FilePath` and `ThumbnailPath` fields to `Item` struct, include them in `List()` and `ByID()` SELECT and Scan

---

## 9. API handler: multipart capture endpoint

**Files:**
- `internal/api/use_cases/item/handler.go` — update `capture()`:
  - Detect `Content-Type: multipart/form-data`
  - Parse the file, validate max size (25 MB), validate MIME type (image/*)
  - Save the file through the storage interface
  - Dispatch `CaptureImage{ItemID, FileRef: filePath, CapturedAt}`
  - Return `{id}` same as text/link

The handler needs access to `storage.Storage` — add it to the handler struct and `NewHandler()` signature.

**Files:**
- `internal/api/use_cases/register.go` — create storage instance, pass to `item.NewHandler`

---

## 10. File serving endpoint

Images need to be served to the browser. The storage's base URL maps to a filesystem directory.

**Files:**
- `internal/api/use_cases/item/handler.go` — add route `GET /files/{path}` (public or protected) that reads from storage and streams the file with the correct `Content-Type`

Or, simpler: serve static files from the storage directory via a middleware or a separate route handler. For self-hosted, `http.FileServer` pointed at the storage dir works.

**Decision:** Use a protected route `GET /files/*` that proxies through the storage interface, so S3 works later.

---

## 11. Thumbnail reactor

Reacts to `ImageCaptured`, generates a thumbnail, dispatches `RecordThumbnail` on the aggregate.

**Files:**
- `use_cases/item/thumbnail/reaction.go` — implements `reaction.Interface`:
  - Subscribes to `ImageCaptured` events
  - Opens the image from storage
  - Decodes it (support JPEG and PNG)
  - Resizes to 200px max width using `golang.org/x/image/draw` (CatmullRom scaling)
  - Encodes as JPEG quality 80
  - Saves thumbnail to storage with key `thumb-{originalKey}`
  - Loads item aggregate, calls `RecordThumbnail(thumbnailPath)`, saves
  - On error: logs warning, returns nil (same as link metadata reactor — don't block the stream)

No tests — same as link_metadata, the reaction logic is tested via integration.

**Files:**
- `cmd/catchup/main.go` — register the thumbnail reactor consumer

---

## 12. WebSocket broadcast for thumbnail updates

When `ThumbnailGenerated` fires, push to the UI so the card updates without a page reload.

**Files:**
- `internal/api/use_cases/register.go` — add `ThumbnailGenerated` to the event types subscribed in `subscribeToMetadataEvents` (rename to `subscribeToPushEvents`). Broadcast `{"type":"thumbnail-generated","data":{"id","thumbnailPath"}}`.

---

## 13. Frontend: image upload in CaptureForm

**Files:**
- `web/src/lib/components/CaptureForm.svelte` — add "image" tab or file-picker button. On file select, call API with multipart upload.
- `web/src/lib/api/client.ts` — add `captureImage(file: File): Promise<string>` function using `FormData`

---

## 14. Frontend: display images in cards and detail

**Files:**
- `web/src/lib/components/ItemCard.svelte` — for image items, show `<img>` thumbnail (or fallback placeholder). Add `thumbnailPath` to Item type.
- `web/src/lib/components/ItemDetail.svelte` — for image items, show full image.
- `web/src/lib/components/Inbox.svelte` — listen for `thumbnail-generated` WebSocket events, update matching item's `thumbnailPath`.
- `web/src/lib/api/client.ts` — add `thumbnailPath` and `filePath` to `Item` interface.

---

## Task order (dependency-driven)

```
 1. Storage interface + disk impl     ◄── no deps
 2. Migration (file columns)           ◄── no deps
 3. Command + event types              ◄── no deps
 4. Aggregate methods                  ◄── depends on 3
 5. Event registration                 ◄── depends on 3
 6. Use case handler                   ◄── depends on 4, 5
 7. Projection                         ◄── depends on 2, 3, 5
 8. Read model (query fields)          ◄── depends on 2
 9. API handler (multipart)            ◄── depends on 1, 6
10. File serving endpoint              ◄── depends on 1
11. Thumbnail reactor                  ◄── depends on 1, 4
12. WebSocket broadcast (thumbnail)    ◄── depends on 3
13. Frontend: upload                   ◄── depends on 9
14. Frontend: display                  ◄── depends on 8, 12
```

Tasks 1–3 can run in parallel. 4–8 depend on 3. 9–10 depend on 1. 11 depends on 1+4. 12 is independent. 13–14 depend on 9+12.

Tests are written alongside each production task, not as separate tasks.