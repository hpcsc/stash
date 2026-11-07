# US-021: Edit a saved note

Editing replaces a note's markdown. The same command is what the extension uses to add a
selection to a note (US-023), so this story owns the whole write path.

## Decisions

- **One command, full-body replacement.** `EditNote{ItemID, Markdown, ExpectedVersion}` records
  `NoteEdited{ItemID, Markdown}`. The event carries the new body, not a delta, because the user
  can change anything in the note.
- **The client sends the version it read.** `ExpectedVersion` is the item's stream version from
  the read model. The aggregate compares it with `Item.Version()` and returns
  `ErrNoteChanged` on a mismatch, so an edit made against a stale copy is refused rather than
  silently overwriting someone else's change. The event store's own optimistic concurrency
  still covers a write that races during the request.
- **Only notes are editable.** Reject when `kind != "note"`, when the item has not been
  captured, and when the note is in the Trash (`state == "dropped"`). A delete is already
  `state == "deleted"`, so it fails the same check.
- **An edit changes content only.** It keeps `state`, `captured_at`, and tags. The list order
  therefore does not move, which matches the story.
- **Images reuse the capture path.** Editing may add images with the same
  `@stash-image-N` placeholders and the same limits as capture. Stored images already in the
  note stay as their `/files/...` keys; only new uploads are posted.
- **Empty is refused.** Whitespace-only markdown returns the existing `ErrNoteEmpty`.
- **Stale edit returns 409.** The handler maps `ErrNoteChanged` to `Conflict`, the same status
  it already uses for `ErrConcurrencyConflict`, with a message telling the client to reload.

## Codebase patterns to follow

- Command and event shape: `server/internal/use_cases/item/command/capture_note.go`,
  `server/internal/use_cases/item/event/note_captured.go`.
- Aggregate rules and `Apply`: `server/internal/use_cases/item/item.go`. `kind` is already set
  to `"note"` on `NoteCaptured`.
- Handler registration: `server/internal/use_cases/item/capture_note/handler.go` plus
  `server/internal/use_cases/register.go`.
- Event registration: `server/internal/use_cases/item/events.go` and
  `server/internal/use_cases/item/items/projection.go` (`EventTypes`).
- Multipart note upload and placeholder replacement:
  `server/internal/api/use_cases/item/handler.go` (`captureUpload`, `captureNote`,
  `maxNoteUploadSize`, `maxMultipartMemory`).
- Read model and `View` interface: `server/internal/use_cases/item/items/query.go`.
- Web client form posts: `captureNote` and `postForm` in `web/src/lib/api/client.ts`.
- Editor lifecycle: `createNoteEditor` in `web/src/lib/editor.ts`; `ItemDetail.svelte` renders
  the note.

## Tasks (dependency order)

### Phase 1 — write model

1. Add `command.EditNote{ItemID uuid.UUID, Markdown string, ExpectedVersion uint64}` next to
   `CaptureNote`.
   - `server/internal/use_cases/item/command/edit_note.go`
2. Add `event.NoteEdited{ItemID uuid.UUID, Markdown string}` with
   `EventType() == ".note-edited"`, using the same namespace prefix as `NoteCaptured`.
   - `server/internal/use_cases/item/event/note_edited.go`
   - register it in `server/internal/use_cases/item/events.go`
3. Add `ErrNoteChanged = errors.New("the note changed after it was read")` and
   `Item.EditNote`. Check, in order: captured, `kind == "note"`, not dropped/deleted, markdown
   not empty, `cmd.ExpectedVersion == i.Version()`. Record `NoteEdited`.
   - `server/internal/use_cases/item/item.go`
4. Handle `*event.NoteEdited` in `Apply`. It changes no aggregate fields; it exists so the
   version advances and a later load sees the new body. If the note body is needed on the
   aggregate later, store it here.
5. Add the use-case handler and register it.
   - `server/internal/use_cases/item/edit_note/handler.go`
   - `server/internal/use_cases/register.go`

### Phase 2 — read model

6. Add `NoteEdited` to `items.EventTypes` and handle it with
   `UPDATE items SET body = $2, version = $3, updated_at = NOW() WHERE id = $1`. Guard the
   update with `AND type = 'note'` so a bad event cannot rewrite a link.
   - `server/internal/use_cases/item/items/projection.go`
7. Read `items.version` into the query `Item` and expose it as `version` in JSON. The client
   needs it for `ExpectedVersion`.
   - `server/internal/use_cases/item/items/query.go` (`itemColumns`, `scanItem`, `Item`)
   - extend the `itemColumns` scan in `ByID` and `List`

### Phase 3 — API

8. Add `route.Protected("PATCH", "/items/{id}", h.editItem)`.
9. Implement `editItem`:
   - parse the id as a UUID;
   - accept `application/json` (`{body, version}`) and `multipart/form-data`
     (`markdown`, `version`, repeated `images`);
   - apply `http.MaxBytesReader` and `ParseMultipartForm` for multipart, as `captureUpload`
     does;
   - validate that `version` parses and is non-zero;
   - dispatch `EditNote` and return `NoteResponse`;
   - map `ErrNoteChanged` to `409` with "the note changed, reload it and try again".
   - `server/internal/api/use_cases/item/handler.go`, `request.go`, `response.go`
10. Extract the note image handling from `captureNote` into one helper that both capture and
    edit call: validate count/size/type, store each image, and replace
    `@stash-image-N`. Return the rewritten markdown and an error. Do this in the same commit as
    `editItem` so the two paths cannot drift.
    - `server/internal/api/use_cases/item/handler.go`
11. Add `editNote(id, markdown, version, images = [])` to the web client: JSON when there are
    no new images, multipart `PATCH` when there are. Reuse `postForm` with a method argument.
    - `web/src/lib/api/client.ts`

### Phase 4 — web editor

12. Let `createNoteEditor` start from existing markdown and resolve stored images. It currently
    hardcodes `defaultValueCtx` to `''`, and a stored key such as
    `![](user/<id>/images/<uuid>.png)` will not load in the editor. Add options for the initial
    markdown plus a way to show a stored key through `fileUrl` and to turn it back into the key
    on save, so the access token is never written into the note.
    - `web/src/lib/editor.ts`
13. Add an edit action to `ItemDetail.svelte` for `item.type === 'note'` when `item.state` is
    `inbox` or `kept`. Swap the rendered markdown for the editor with Save and Cancel. On
    success, replace the item in place rather than refetching; on `409`, keep the editor open
    and show the reload message.
    - `web/src/lib/components/ItemDetail.svelte`
    - the Inbox, Library, and Trash containers that own the selected item
14. Keep the existing create-note flow untouched: `CaptureForm.svelte` and
    `Inbox.svelte` continue to call `captureNote`.

### Phase 5 — tests and docs

15. Server tests: aggregate rejects a non-note, a dropped note, empty markdown, and a stale
    version; projection updates only a note; handler accepts JSON and multipart, returns `409`
    on a stale version, and rewrites image placeholders.
    - `server/internal/use_cases/item/item_test.go`
    - `server/internal/use_cases/item/items/*_test.go`
    - `server/internal/api/use_cases/item/*_test.go`
16. Web tests: client sends the version and switches between JSON and multipart; `ItemDetail`
    enters and leaves edit mode; a stale save keeps the user's text.
17. Update the event model and the story map for `NoteEdited`.
    - `docs/event-model-stash.md`, `docs/event-model-stash.emod`
    - `user-stories/stash-story-map.html`
