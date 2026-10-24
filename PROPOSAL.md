# Stash — Project Proposal

A self-hostable **capture inbox**: save text, images, links, and files from any app on
your phone or desktop, then sort them into tags or discard them. Event-sourced, with the
event store and projections both in PostgreSQL.

---

## 1. Problem

While browsing or chatting on a phone, you hit a piece of text, an image, a link, or a
file you want to keep. Existing tools each miss part of this:

- **Raindrop** nails the *inbox → sort → drop* workflow, but it is link/bookmark-first;
  capturing arbitrary text or files is clunky.
- **Google Keep** captures text fine but has no inbox → triage stage; everything lands as
  a flat note you must file immediately.
- **Readwise Reader** has a triage inbox but is built around articles and highlights, is
  article-shaped, and is paid.
- **Memos** is text-first and self-hosted, but a flat timeline with no "unsorted awaiting
  a decision" stage.

**Stash** fills the gap: a capture inbox (GTD-style) that accepts *all four* content types,
triages them uniformly, and is self-hostable.

## 2. Goals

- Capture **text, image, link, and file** from any Android app via the share sheet, and
  from a desktop browser via an extension.
- Everything lands in an **Inbox**, unsorted, with source app/URL recorded.
- **Triage**: keep (into tags) or drop (into trash). Fast, from phone or web.
- **Search** across saved text and links.
- **Self-hosted** by default, with a cloud deployment path behind the same interface.
- **Event-sourced**: an immutable capture log, with the event store and projections both
  in PostgreSQL.
- **v1 platforms**: Android + Web, plus a desktop browser extension for capture.

## 3. Non-goals (v1)

- iOS client (Flutter makes it possible later; not in v1).
- Multi-user collaboration / sharing (single-user, but the schema stays `user`-scoped).
- Full-page archival / offline reading of saved links.
- AI tagging/summarization (a possible later phase, not v1).
- Video capture (large files are out of scope until size limits are decided).
- Native desktop app (macOS/Windows/Linux) — desktop triage is the Flutter web app, and
  desktop capture is the browser extension (D12).

## 4. Core workflow

```
capture (share sheet / extension) ──► INBOX ── keep ──► LIBRARY (tagged, searchable)
                                         └── drop ──► TRASH (recoverable, emptyable)
```

1. On the phone, tap **Share** in any app → choose **Stash**. On desktop, use the
   extension to save selection / page / image.
2. Stash records the item (type, body, source app/URL) into the **Inbox**.
3. Later, in the Inbox: **Keep** (assign one or more tags) or **Drop** (→ Trash).
4. Kept items are grouped/filterable by tag and full-text searchable.
5. Trash items can be restored or permanently deleted.

## 5. Capture types

| Type | Source | Android mechanism | Stored as |
|---|---|---|---|
| `text` | any app's "share text" | `ACTION_SEND` + `EXTRA_TEXT` | text body |
| `link` | browser/share of a URL | `ACTION_SEND` + `EXTRA_TEXT` (URL) | URL + fetched title/description |
| `image` | gallery/chat photo | `ACTION_SEND` + `EXTRA_STREAM` (`image/*`) | file + thumbnail |
| `file` | any file share | `ACTION_SEND` + `EXTRA_STREAM` (`*/*`) | file + name/size/MIME |

`source_app` (the sending package) and `source_url` (for text/link) are captured so you
can trace an item back to where it came from.

On desktop, a **browser extension** covers the same types — selected text (`text`), the
current page (`link`), and a right-clicked image (`image`) — posting to the same API.

## 6. Architecture

```
┌─────────────────────────────┐        ┌───────────────────────────────────────────┐
│  Flutter app (Android+Web)  │  JSON  │  Go service (event-sourced, Postgres)      │
│  - share target (Android)   │ ─────► │  cmd/api      HTTP API (chi, token auth)   │
│  - Inbox/Library/Trash UI   │ HTTPS  │  cmd/catchup  projections + reactions       │
│  - offline queue (drift)    │        │  use_cases/account, item, trash, tag       │
└─────────────────────────────┘        │  events table (append-only) + read models  │
┌─────────────────────────────┐        │  storage: Disk / S3 (file bytes)          │
│  Browser extension          │  JSON  └───────────────────────────────────────────┘
│  (Chrome, then Firefox)     │ ─────►
│  text / link / image        │ HTTPS
└─────────────────────────────┘
```

The backend is scaffolded from the `go-event-sourcing-service` template (with the
`use_cases` naming) and keeps its shape:

- **Write path** — the API dispatches commands; a handler loads the aggregate from its
  stream, records events, and appends them to the `events` table (optimistic concurrency).
- **Read path** — `cmd/catchup` runs a consumer loop that wakes on `LISTEN/NOTIFY` and
  projects events into read-model tables. Read models are *eventually consistent*: a
  captured item appears in the Inbox a moment after the command, when the projector runs.
- **Files** — a `storage` behaviour (Disk for self-hosted, S3 for cloud) holds the bytes;
  the event stores only the `file_path` reference. This is new to the template.

**Client** — one Flutter codebase targeting Android and Web. Android registers a native
share-target (`receive_sharing_intent`) so Stash appears in the system share sheet; a
drift-backed offline queue posts to the API when connectivity returns. Web runs the same
triage/search UI.

**Auth** — token middleware (from the template) for the Flutter app and the extension;
sign-in and sign-out are recorded as `SignedIn` / `SignedOut` events.

## 7. Data model

**Event store** (one append-only table, from the template):

```sql
CREATE TABLE events (
    sequence       BIGSERIAL PRIMARY KEY,
    transaction_id XID8 NOT NULL DEFAULT pg_current_xact_id(),
    id             UUID NOT NULL UNIQUE,
    stream_id      TEXT NOT NULL,
    version        BIGINT NOT NULL,
    type           TEXT NOT NULL,
    data           JSONB NOT NULL,
    metadata       JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT events_stream_version UNIQUE (stream_id, version)
);
```

**Streams & events** — one stream per aggregate (`account-<userId>`, `item-<uuid>`,
`trash-<userId>`, `tag-<uuid>`):

```
account: SignedIn  SignedOut
item:    TextCaptured  LinkCaptured  ImageCaptured  FileCaptured  LinkMetadataFetched  ThumbnailGenerated  ItemKept  ItemTagged  ItemDropped  ItemRestored  ItemDeleted
trash:   TrashEmptied
tag:     TagCreated  TagRenamed
```

Event types are namespaced (`stash.text-captured`, `stash.tag-created`, …).

**Read models** (projections written by `cmd/catchup`):

```text
items
  id, type (text|image|link|file), body, source_url, source_app,
  file_path, file_name, file_size, mime_type,
  state (inbox|kept|dropped), version, created_at, updated_at
  -- pg_trgm GIN index on body for full-text search

tags
  id, name

item_tags            -- many-to-many, from ItemTagged
  item_id, tag_id
```

- `state` is a projection of the item's latest event, not a column the write path sets.
- `Library` = `kept`; `Trash` = `dropped`; `Inbox` = `inbox`.
- One attachment per item in v1 (an image item holds one file). Multi-file per item is a
  later phase.

## 8. API (v1)

Commands dispatch to the domain and append events; reads query the read models:

```
POST /api/v1/items                Capture (multipart for image/file; JSON for text/link)
POST /api/v1/items/{id}/keep      Keep (move to Library; assign tags)
POST /api/v1/items/{id}/retag     Retag (change tags)
POST /api/v1/items/{id}/drop      Drop
POST /api/v1/items/{id}/restore   Restore
POST /api/v1/items/{id}/delete    Delete (permanent, from Trash)
POST /api/v1/trash/empty          Empty trash
POST /api/v1/tags                 CreateTag
POST /api/v1/tags/{id}/rename     RenameTag

GET  /api/v1/items                list, filters ?state=&tag=&q=
GET  /api/v1/items/{id}
GET  /api/v1/tags
```

Auth is the token middleware from the template; sign-in and sign-out are recorded as
`SignedIn` / `SignedOut` events on the account aggregate.

## 9. Triage UX

Both surfaces — the Flutter web app and the Flutter Android app — are one codebase and
offer the same three views and the same keep/drop actions.

- **Inbox** — list of unsorted items. Swipe right = **Keep**, swipe left = **Drop**;
  tap opens detail with the same two actions plus tag assignment.
- **Library** — kept items, grouped/filtered by tag, full-text search.
- **Trash** — dropped items with **Restore** and **Empty**.

## 10. Deployment

- **Self-hosted**: `docker compose` — the `api` and `catchup` processes, PostgreSQL, the
  `migrate` container, and a file volume. The phone reaches it over **Tailscale** (no
  public exposure, no TLS hassle).
- **Cloud**: same images on a VPS/Fly.io + managed Postgres + an S3-compatible bucket;
  swap the storage adapter via config.

## 11. Decision ledger

| # | Decision | Rationale |
|---|---|---|
| D1 | Build "Stash" rather than adopt an existing tool | No existing tool does text+image+link+file capture with an inbox→triage flow on Android |
| D2 | v1 platforms = Android + Web + desktop capture | Phone and desktop are both capture surfaces; web is the comfortable triage surface |
| D3 | Capture types = text, image, link, file | User's stated scope |
| D4 | Workflow = Inbox → keep (tag) / drop (trash) | Raindrop-style triage, the core requirement |
| D5 | Client = Flutter (Android + Web, one codebase) | Cross-platform; gets iOS later for free; native share-target via plugin |
| D6 | Backend = Go, event-sourced service on Postgres | Adopts the user's existing Go event-sourcing template |
| D7 | Storage behind an interface (Disk / S3) | Self-hosted and cloud from the same code |
| D8 | Name = Stash | Names the behaviour (grab and stash), one plain word |
| D9 | Categories = tags (many-to-many) | Flatter and more flexible than folders |
| D10 | Drop = soft-delete to Trash | Recoverable, then emptyable |
| D11 | Event sourcing: store + projections both in Postgres, async catchup consumer | No second event store; template's `LISTEN/NOTIFY` loop; read models eventually consistent |
| D12 | Desktop triage = Flutter web; desktop capture = browser extension (text + link + image); no native desktop app | Web triage is already the Flutter app; the extension is the desktop analogue of the share sheet |
| D13 | File size cap = 25 MB; video excluded | Keeps storage bounded; over-cap files are rejected with a clear message |
| D14 | Adopt `go-event-sourcing-service` template; module `github.com/hpcsc/stash`; event namespace `stash` | Reuse the proven event store, subscription, and consumer loop |
| D15 | Package naming `use_cases` (not `usecase`) | User's convention; template renamed to match |

## 12. Milestones

| Milestone | Scope | Exit criterion |
|---|---|---|
| **M0** | This proposal | Decisions D1–D15 confirmed |
| **M1** | Scaffold from template + implement `item` and `tag` domains | Aggregates, events, commands, use cases, projections (items, tags, item_tags), migrations, API handlers; tests green |
| **M2** | Flutter app (Android + Web) | auth, Inbox/Library/Trash screens working against the API, running in a browser and on a device/emulator |
| **M3** | Android share target + offline queue | share text/image/link/file from any app → appears in Inbox; offline queue syncs |
| **M4** | Browser extension (Chrome, then Firefox) | save selection (text), save page (link), save image (right-click) → all appear in Inbox |
| **M5** | Polish | image thumbnails, tag management, search UX, error/retry states |
| **M6** | Deployment | `docker compose` + Tailscale documented; cloud path documented |

## 13. Definition of done (v1)

1. From Android, sharing text, an image, a link, or a file from another app lands it in the Inbox.
2. From a desktop browser, the extension saves selected text, a page link, or an image into the Inbox.
3. The same Inbox is visible and triageable from the web app.
4. **Keep** tags an item and moves it to Library; **Drop** moves it to Trash; Trash restores and empties.
5. Full-text search finds text and links by content.
6. Capture works offline and syncs when connectivity returns.
7. Self-hosted via `docker compose`; data and files survive a container restart.

## 14. Edge cases & open questions

- **Large files / video** — 25 MB cap, video excluded (D13); over-cap files are rejected with a clear message.
- **Duplicates** — share the same text/URL twice → allow, warn, or dedupe? (Proposal: allow, surface "already saved" later.)
- **Multi-image share** — one share containing several photos → one item per file (v1), or one item with multiple attachments (later).
- **Link metadata** — fetch title/description as a reaction (or at capture time) so the client stays simple.
- **Offline** — queue in drift; retry with backoff; show pending count.
- **Unknown source app** — `source_app` is null when the OS doesn't report it; don't block capture.
- **Auth model** — single-user for now, but the token middleware and `user`-scoped schema keep multi-user possible.
- **Eventual consistency** — a capture shows up in the Inbox a moment after the command, once `cmd/catchup` projects it.

---

*All decisions D1–D15 confirmed — ready for M1.*
