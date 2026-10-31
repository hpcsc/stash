# US-009: Warn on duplicate captures

## Decisions

- **Duplicate = same body for the same type.** Text matches text (`type = 'text' AND
  body = X`); a link matches a link (`type = 'link' AND body = URL` — a link's body is its
  URL). Exact match: no normalization, no fuzzy matching.
- **Read model, not an event.** A duplicate is a fact about the current `items` table, so
  the API checks the read model after a capture, the way Search reads it. No new event or
  command.
- **Still capture, then warn.** The capture always succeeds; the response carries
  `duplicate` and `duplicateState` so the client can show a notice without a second
  request. A read error means "no duplicate" — the item is already saved, so the capture
  must not fail.
- **Every state counts.** A duplicate in the Inbox, the Library, or the Trash warns, and
  the notice says where (`duplicateState` is `inbox` | `kept` | `dropped`).

## Codebase patterns to follow

- The handler's `ItemReader` gains one method; the unit test's `fakeReader` implements it.
- Text and link captures both return `{id, duplicate, duplicateState}`.

## Tasks (dependency order)

### Phase 1 — read model

1. Add `items.Query.Existing(ctx, itemType, body)` returning the existing item's id and state.
   - `internal/use_cases/item/items/query.go`.

### Phase 2 — API

2. Add `Existing` to the `ItemReader` interface; return `{id, duplicate, duplicateState}`
   for text and link captures.
   - `internal/api/use_cases/item/handler.go`, `internal/api/use_cases/item/response.go`.

### Phase 3 — web

3. `captureText` and `captureLink` return the duplicate info.
   - `web/src/lib/api/client.ts`.
4. Show a warning banner in the Inbox when a capture is a duplicate.
   - `web/src/lib/components/Inbox.svelte`.

### Phase 4 — tests

5. Handler unit tests for duplicate and non-duplicate text and link captures.
   - `internal/api/use_cases/item/handler_test.go`.
6. Read-model integration test for `Existing`.
   - `internal/use_cases/item/items/projection_test.go`.
7. Web client tests for the new return shape.
   - `web/src/lib/api/client.test.ts`.
