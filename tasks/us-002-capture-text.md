# US-002: Capture text into the Inbox

## Decisions

- **`capturedAt` stays on the event.** "Newest first" needs a sortable time, and the
  projection cannot see the event store's `created_at` (the `domain.Event` does not carry
  it). The item events keep their timestamp; the identity events dropped theirs because
  sign-in has no ordering requirement.
- **No user scoping yet** — single-user v1; `TextCaptured` carries no subject, matching the
  event model.
- **Item stream** `item-<itemId>`; the API generates the UUID and the capture time.
- **Minimal items table** — `type`, `body`, `source_app`, `source_url`, `captured_at`;
  `state` and the file columns arrive with later stories.

## Tasks

### Phase 1 — domain
1. Add the `TextCaptured` event.
2. Add the `CaptureText` command.
3. Add the `Item` aggregate (`CaptureText`, `CapturedOnce`) + tests.
4. Add the repository.
5. Add the capture-text command handler.
6. Wire events + handlers into `internal/use_cases/register.go`.

### Phase 2 — read model
7. Add the `items` table migration.
8. Add the items projection (`internal/use_cases/item/items`).
9. Add the items query (list newest-first, by ID).
10. Register the projection in `cmd/catchup`.

### Phase 3 — API
11. `POST /items` — capture text.
12. `GET /items` — inbox, newest first.
13. `GET /items/{id}` — item detail.
14. Wire routes.

### Phase 4 — tests
15. API handler tests.
