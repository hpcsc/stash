# US-019 / US-020: Markdown notes

US-019 merges text and notes into one markdown item; US-020 adds remote image URLs. They
share one implementation and one task list.

## Decisions

- **Markdown is the text.** A note holds one markdown string, not a list of `text`/`image`
  blocks. Formatting (`##`, `**bold**`, `[label](url)`) is the markdown source itself, rendered
  by the client. There is no structured block model on the server.
- **Text and note are one kind of item.** `TextCaptured` folds into `NoteCaptured`; the note
  event becomes `{ ItemID, Markdown, CapturedAt }`. Not live, so the rewrite is in place —
  no `schema_version` bump, no upcaster, local DB reset (same call as US-018).
- **Images are inline markdown.** A local image uploads to `user/<id>/...` and is referenced
  `![](path)`; a remote image is `![](https://...)` and is never stored. Drop `item_images` and
  `note_blocks`.
- **Drop `sourceApp`/`sourceUrl`.** No client ever sends them (the extension posts `{body}`,
  the web posts `{body}`), and a single source URL cannot describe a note assembled from many
  pages and images. Provenance belongs in the markdown as links.
- **Drop the duplicate warning for text/note; keep it for links.** US-009 narrows to "warn on
  duplicate links" — a URL is an exact key, a markdown document is not.
- **No thumbnails for notes in v1.** The thumbnail reactor stops handling `NoteCaptured`; a
  markdown-aware pass that finds `![](/files/...)` refs is a follow-on. Standalone image items
  keep their thumbnails.
- **Milkdown is the editor.** Its ProseMirror input rules render `## `, `**`, `-`, `>`,
  `` ` `` and `[label](url)` live as the syntax completes.
- **Render markdown sanitized.** DOMPurify on the rendered HTML; remote images get
  `referrerpolicy="no-referrer"`; relative `/files/...` refs get the session token via
  `fileUrl`, absolute `http(s)` refs pass through untouched.

## Codebase patterns to follow

- The capture lives in `item.Item.CaptureText`/`CaptureNote` and the `Apply` switch on
  `TextCaptured`/`NoteCaptured` (`server/internal/use_cases/item/item.go`); events register in
  `server/internal/use_cases/item/events.go`.
- The block model to remove: `note.Block`/`TypeText`/`TypeImage` (`item/note/note.go`),
  `note_blocks` JSONB + `item_images` inserts (`item/items/projection.go`), and
  `readNoteBlocks`/`attachImages`/`blocksWithImages` (`item/items/query.go`).
- The note multipart upload (`captureNote`/`captureUpload` in the API handler) already uploads
  images and references them; the markdown path keeps the "upload then rewrite refs" idea, just
  with `![](ref)` instead of image blocks.
- `thumbnail/reaction.go` generates thumbnails for `ImageCaptured` and `NoteCaptured`; the
  `NoteCaptured` half is the deferred piece.
- Web: `CaptureForm.svelte` is a hand-rolled `contenteditable` and `note.ts` walks its DOM;
  `Inbox.svelte` routes URL → link, text → text, images → note; `ItemDetail.svelte` renders
  notes from blocks.

## Tasks (dependency order)

### Phase 1 — event model: one markdown note
1. Fold `TextCaptured` into `NoteCaptured`: the note event becomes
   `{ ItemID, Markdown, CapturedAt }`. Remove `CaptureText` command and event; `NoteCaptured`
   drops `Blocks` for `Markdown`.
   - `server/internal/use_cases/item/event/note_captured.go`, `text_captured.go` (remove)
   - `server/internal/use_cases/item/command/capture_note.go`, `capture_text.go` (remove)
   - `server/internal/use_cases/item/item.go` (`CaptureNote`, `Apply`)
   - `server/internal/use_cases/item/events.go`
2. Drop `sourceApp`/`sourceUrl` from command, event, request, projection, and query.

### Phase 2 — read model
3. Projection writes the markdown to `body`; drop the `note_blocks` and `item_images` inserts.
   - `server/internal/use_cases/item/items/projection.go`
4. Query drops the `note` branches and `readNoteBlocks`/`attachImages`/`blocksWithImages`;
   returns `body` as the markdown.
   - `server/internal/use_cases/item/items/query.go`
5. Rewrite migrations: drop `note_blocks`, `item_images`, `source_app`, `source_url`.
   - `server/migrations/20261024120000_create-items-table.up.sql` (+ `.down.sql`)
   - `server/migrations/20261029120000_add-notes-and-images.up.sql` (+ `.down.sql`)
6. Stop the thumbnail reaction from handling `NoteCaptured` (deferred markdown-aware pass).
   - `server/internal/use_cases/item/thumbnail/reaction.go`

### Phase 3 — API
7. Handler: text capture and note upload both produce one markdown note; local images upload
   and their refs are rewritten to `/files/...`; remote URLs pass through.
   - `server/internal/api/use_cases/item/handler.go`, `request.go`, `response.go`
8. Remove the text duplicate check; keep the link duplicate warning (US-009).
   - `server/internal/api/use_cases/item/handler.go` (`captureResponse`)
   - `server/internal/use_cases/item/items/query.go` (`Existing`)

### Phase 4 — web editor (Milkdown)
9. Replace the `contenteditable` editor in `CaptureForm.svelte` with Milkdown; live input
   rules.
   - `web/src/lib/components/CaptureForm.svelte`, `web/package.json`
10. Local image drop/paste: blob preview → upload on save → rewrite the ref to `/files/...`.
11. Remote image URL insertion, rendered inline.
12. Serialize markdown on save (`getMarkdown()`) and send it as a note; `Inbox.svelte` routes
    everything text-ish to the note path.
    - `web/src/lib/note.ts` (replace), `web/src/lib/api/client.ts`
    - `web/src/lib/components/Inbox.svelte`

### Phase 5 — rendering
13. Render markdown sanitized in `ItemDetail.svelte` and `ItemCard.svelte`; `/files/...` refs
    get the session token, absolute `http(s)` as-is.

### Phase 6 — tests & docs
14. Tests: markdown renders and search still finds it; image drag and image URL; no duplicate
    warning for text, link duplicate still warns.
15. Update `docs/event-model-stash.md` and `docs/event-model-stash.emod` for the
    `TextCaptured` → `NoteCaptured` merge and the markdown body.
