# US-018: Give each user a private stash

## Decisions

- **The user is the stream, not a field.** Every event stream that holds user data is
  prefixed with the user ID: `user/<id>/item-<uuid>`, `user/<id>/tag-<uuid>`,
  `user/<id>/trash`. The event payloads and the `events` table are untouched — no `user_id`
  column, no per-payload field, no `schema_version` bump, no upcaster. `identity-<subject>`
  keeps its shape: it already identifies one user and is never projected or broadcast.
- **`user` is the noun, "own" is the verb.** The domain type is `domain.UserID`, the column
  is `user_id`, the stream segment is `user/`. "Owner" appears only in prose and in the
  authorization error. `subject` stays as the token-claim name at the auth boundary; the
  conversion to `UserID` happens once, in the middleware.
- **The user rides on the command, not in the context.** `domain.Command` gains `Actor`, set
  by the HTTP handler from the session. The use-case layer then has no hidden parameter, so a
  missing user is a compile error rather than a runtime 500.
- **Scoping is a view, not a filter.** `Repository.For(userID)` and `Query.For(userID)` return
  a view whose `Load`/`Save`/`List`/`ByID`/`Existing` are the only exported operations, and
  the unscoped ones are unexported. A handler holding the view physically cannot name another
  user's stream, so there is no `WHERE user_id = $1` to forget.
- **Async work derives the user from the stream.** Reactions, projections, and the push
  broadcaster have no session, only `*domain.Event`, so `domain.UserOf(evt.StreamID)` is
  their entry point. This is why the user lives in the stream ID rather than in the payload:
  the payload is per event type, the stream is per aggregate.
- **Not live, so the migration is a rewrite.** Edit the existing item and tag migrations in
  place instead of adding a backfill migration with a defaulted column. There is no data to
  preserve and no client to stay compatible with.
- **RLS is a follow-on, not part of this story.** It needs a non-owner application role (the
  compose file currently runs `migrate` and `api` as the same `DB_USERNAME`, so `api` owns
  the tables) and `set_config` inside every request transaction. Both are only worth doing
  once application-level scoping exists.

## Codebase patterns to follow

- `middleware.WithSubject` / `SubjectFromContext` already put the Google `sub` on the request
  context; only `/auth/signout` reads it today.
- Each aggregate owns its stream name: `item.StreamID`, `tag.StreamID`, `identity.StreamID`,
  and `trash.StreamID` (a constant today — the trash is currently shared by everyone).
- `store.OfEvents.Load/Save` take a `streamID string`, so scoping lives one level above the
  event store.
- `Projection.Handle(ctx, tx, evt)` and `reaction.Interface.Handle(ctx, evt)` both receive
  the whole event, including `StreamID`.
- `items.Query` uses bare `pool.Query`/`QueryRow`, so the scoping has to be written into the
  SQL. RLS would need a transaction; this story does not.

## Tasks (dependency order)

### Phase 1 — vocabulary and stream layout
1. `domain.UserID`, `domain.UserOf(streamID)`, and a single stream-ID builder per aggregate.
   - `server/internal/domain/user.go` (new)
2. `domain.Command` gains `Actor`; `NewCommand` requires it.
   - `server/internal/domain/command.go`
3. Make each user-data stream take a `domain.UserID`.
   - `server/internal/use_cases/item/item.go`
   - `server/internal/use_cases/tag/tag.go`
   - `server/internal/use_cases/trash/repository.go` (constant → builder)

### Phase 2 — command path
4. HTTP handlers pass the authenticated user when they dispatch.
   - `server/internal/api/middleware/auth.go` (expose `UserFromContext`)
   - `server/internal/api/use_cases/item/handler.go`
   - `server/internal/api/use_cases/tag/handler.go`
5. Repositories and read models expose `For(userID)` and unexport the unscoped operations.
   - `server/internal/use_cases/item/repository.go`
   - `server/internal/use_cases/item/items/query.go`
   - `server/internal/use_cases/tag/repository.go`
   - `server/internal/use_cases/tag/tags/query.go`
   - `server/internal/use_cases/trash/repository.go`
6. Command handlers scope with `cmd.Actor`.
   - the handlers under `server/internal/use_cases/item/*/handler.go`
   - `server/internal/use_cases/tag/*/handler.go`
   - `server/internal/use_cases/trash/*/handler.go`

### Phase 3 — async path
7. Reactions derive the user from the event stream before loading an aggregate.
   - `server/internal/use_cases/item/link_metadata/reaction.go`
   - `server/internal/use_cases/item/thumbnail/reaction.go`
   - `server/internal/use_cases/trash/empty_trash_reactor/reaction.go`
8. Projections write the user onto the read model from the event stream.
   - `server/internal/use_cases/item/items/projection.go`
   - `server/internal/use_cases/item/item_tags/projection.go`

### Phase 4 — read model
9. Add `user_id` to items and tags, and make tag names unique per user.
   - `server/migrations/20261024120000_create-items-table.up.sql` (+ `.down.sql`)
   - `server/migrations/20261025130000_create-tags-tables.up.sql` (+ `.down.sql`)
10. Scope every read: list, by-id, duplicate check, search, trash, and tag list.
    - `server/internal/use_cases/item/items/query.go`
    - `server/internal/use_cases/tag/tags/query.go`

### Phase 5 — files and realtime
11. Prefix storage keys with the user and stop trusting the download URL.
    - `server/internal/api/use_cases/item/handler.go` (`images/` and `files/` keys,
      `serveFile` wildcard)
12. Make the WebSocket per user.
    - `server/internal/api/websocket/handler.go`
    - `server/internal/api/websocket/hub.go`
    - `server/internal/api/use_cases/register.go` (push subscription)

### Phase 6 — tests and docs
13. Tests: two users with the same tag name; cross-user read, change, delete, and download all
    fail; the duplicate warning is per user; realtime isolation.
14. Update the model and the isolation notes.
    - `docs/event-model-stash.md`
    - `docs/event-model-stash.emod`
    - `docs/websocket.md`
