# Event Model: Stash

A self-hostable capture inbox: save text, links, images, and files from any app, then keep
them into tagged libraries or drop them into a recoverable trash. Event-sourced, with the
event store and projections both in PostgreSQL.

Model: `event-model-stash.emod` · Diagram: `event-model-stash.drawio`

## Actors

| Actor | Type | Description |
|---|---|---|
| User | Human | The single owner, capturing from web, Android, or a browser extension |
| Google | External system | OpenID Connect identity provider |
| Target page | External system | The page a captured link points at; source of title/description (fetched, no callback) |

## Business Processes

Six processes. Each is a timeline of events, the commands that cause them, the rules that
refuse them, the views that back the screens, and the automations that bridge them.

### Process 1 — Sign in / sign out

Timeline: `SignedIn` → `SignedOut`

| Event | Description | Key data |
|---|---|---|
| `SignedIn` | The user authenticated with Google | subject, provider |
| `SignedOut` | The user signed out | subject |

| Command | Actor | Produces | Refused when |
|---|---|---|---|
| `SignIn` | User | `SignedIn` | empty subject, empty provider |
| `SignOut` | User | `SignedOut` | `NotSignedIn` |

Google is the identity provider; the provider is interchangeable. "Refuse while signed out"
is middleware, not an event.

### Process 2 — Capture an item

Timeline: `TextCaptured` / `LinkCaptured` / `ImageCaptured` / `FileCaptured` / `NoteCaptured` → (`LinkMetadataFetched`, `ThumbnailGenerated`, `ImageThumbnailGenerated`)

| Event | Description | Key data |
|---|---|---|
| `TextCaptured` | Text saved to the Inbox | itemId, body, sourceApp?, sourceUrl? |
| `LinkCaptured` | Link saved to the Inbox | itemId, url, sourceUrl? |
| `ImageCaptured` | Image saved to the Inbox | itemId, fileRef |
| `FileCaptured` | File saved to the Inbox | itemId, fileRef, fileName, fileSize, mimeType |
| `NoteCaptured` | Note with ordered text and image blocks saved to the Inbox | itemId, blocks[] |
| `LinkMetadataFetched` | Link title/description fetched | itemId, title?, description? |
| `ThumbnailGenerated` | Image thumbnail generated | itemId, thumbnailRef |
| `ImageThumbnailGenerated` | Thumbnail generated for one image in a note | itemId, imageId, thumbnailRef |

| Command | Actor | Produces | Refused when |
|---|---|---|---|
| `CaptureText` | User | `TextCaptured` | `CapturedOnce` |
| `CaptureLink` | User | `LinkCaptured` | `CapturedOnce` |
| `CaptureImage` | User | `ImageCaptured` | `CapturedOnce` |
| `CaptureFile` | User | `FileCaptured` | `CapturedOnce`, `FileTooLarge`, `VideoRejected` |
| `CaptureNote` | User | `NoteCaptured` | `CapturedOnce`, `NoteEmpty`, `TooManyImages`, `NoteTooLarge`, `InvalidNoteBlock`, `FileTooLarge`, `VideoRejected` |
| `RecordLinkMetadata` | reactor | `LinkMetadataFetched` | `MetadataRecordedOnce` |
| `RecordThumbnail` | reactor | `ThumbnailGenerated` | `ThumbnailRecordedOnce` |
| `RecordImageThumbnail` | reactor | `ImageThumbnailGenerated` | `ThumbnailRecordedOnce`, `ImageNotAttached` |

The capture types are separate events, not a `type` discriminator on one event — the
payloads diverge (body vs url vs file vs blocks), so a discriminator would be a bag of
mutually-exclusive nullable fields. The `Item` aggregate is one: the only uniform thing
across types is the location state machine, which is the aggregate's invariant. The type
lives in the event type, not a field.

A note carries its whole document in one event: ordered `text` and `image` blocks. The
text is the note body; images keep their position in the block list. One note is one
triage unit, so keep, drop and tag act on the note as a whole. Images are captured as part
of the note rather than as separate items, which is why the composer holds them until the
user saves.

`FileTooLarge` (25 MB), `VideoRejected` and `NoteTooLarge` (100 MB total) are
command-input validation, decided from the command payload rather than the item's history.
A note has at most 10 images and must contain text or an image.

| Trigger | Reactor | Command | Result |
|---|---|---|---|
| `LinkCaptured` | `LinkMetadataReactor` | `RecordLinkMetadata` | `LinkMetadataFetched` |
| `ImageCaptured` | `ThumbnailReactor` | `RecordThumbnail` | `ThumbnailGenerated` |
| `NoteCaptured` | `ThumbnailReactor` | `RecordImageThumbnail` × N | `ImageThumbnailGenerated` × N |

A link whose title/description cannot be fetched is still saved with its URL — the absence
of `LinkMetadataFetched` is not a failure.

### Process 3 — Keep to Library

Timeline: `ItemKept` → `ItemTagged` (retag: `ItemTagged` alone)

| Event | Description | Key data |
|---|---|---|
| `ItemKept` | Item moved to the Library | itemId |
| `ItemTagged` | Item's tags set to a set | itemId, tags |

| Command | Actor | Produces | Refused when |
|---|---|---|---|
| `KeepItem` | User | `ItemKept` (+ `ItemTagged`) | `KeepFromInbox` |
| `TagItem` | User | `ItemTagged` | `TagFromLibrary` |

Location and tags are orthogonal. `ItemKept` is pure location; `ItemTagged` is the single
tag fact, produced by both `KeepItem` (initial tags) and `TagItem` (later changes) —
cohesion, not false equivalence. A brand-new tag name at keep time runs `CreateTag` first,
then `ItemTagged` carries the returned `tagId`.

### Process 4 — Drop to Trash & recover

Timeline: `ItemDropped` → (`ItemRestored` / `ItemDeleted`) → `TrashEmptied` → `ItemDeleted` × N

| Event | Description | Key data |
|---|---|---|
| `ItemDropped` | Item moved to the Trash | itemId |
| `ItemRestored` | Item returned from Trash | itemId, restoredTo (inbox\|library) |
| `ItemDeleted` | Item permanently gone | itemId |
| `TrashEmptied` | Trash emptied in one action | itemIds |

| Command | Actor | Produces | Refused when |
|---|---|---|---|
| `DropItem` | User | `ItemDropped` | `DropFromInboxOrLibrary` |
| `RestoreItem` | User | `ItemRestored` | `RestoreFromTrash` |
| `DeleteItem` | User | `ItemDeleted` | `DeleteFromTrash` |
| `EmptyTrash` | User | `TrashEmptied` | `TrashNotEmpty` |

`TrashEmptied` is the single user-facing fact on the `Trash` aggregate; it carries the
explicit `itemIds` resolved at command time. A reactor fans it out to one `DeleteItem` per
item, so each item's own stream ends in `ItemDeleted` and the `Item` aggregate stays
self-contained. The fan-out is safe to run twice: a repeated `DeleteItem` is refused by
`DeleteFromTrash`.

| Trigger | Reactor | Command | Result |
|---|---|---|---|
| `TrashEmptied` | `EmptyTrashReactor` | `DeleteItem` × N | `ItemDeleted` × N |

`restoredTo` is derived by the aggregate from history (inbox if never kept, library if
kept) and recorded on the event — a derived field, not a type discriminator.

### Process 5 — Organize with tags

Timeline: `TagCreated` → `TagRenamed`

| Event | Description | Key data |
|---|---|---|
| `TagCreated` | Tag created | tagId, name |
| `TagRenamed` | Tag renamed | tagId, name |

| Command | Actor | Produces | Refused when |
|---|---|---|---|
| `CreateTag` | User | `TagCreated` | `TagNameTaken` |
| `RenameTag` | User | `TagRenamed` | `TagNameTaken` |

`TagNameTaken` (case-insensitive) is set-level uniqueness: one tag's stream cannot see a
sibling's name, so enforcement needs a name index outside the per-tag stream. Items
reference tags by `tagId`, so rename is a single event and the Library resolves names via a
join.

### Process 6 — Search

No commands or events. Search reads the captured text and link content (with the fetched
title/description) from the read model, updating as the user types, with a "nothing found"
empty state.

## Read Models (Views)

| View | Subscribes to | Purpose | Read by |
|---|---|---|---|
| `ItemsView` | 5 capture events | Captured content, type, source, note blocks and images | Inbox item detail |
| `ItemStateView` | 4 location events | Triage state (inbox/library/trash) | Drop / Restore / Delete / Empty triggers |
| `ItemTagsView` | `ItemTagged` | Item-to-tag mapping | Library item |
| `TagsView` | `TagCreated`, `TagRenamed` | Tag names | Tags screen |
| `LinksToEnrichView` | `LinkCaptured`, `LinkMetadataFetched` | Links still needing metadata | LinkMetadataReactor |
| `ImagesToThumbnailView` | `ImageCaptured`, `ThumbnailGenerated`, `NoteCaptured`, `ImageThumbnailGenerated` | Images still needing thumbnails | ThumbnailReactor |
| `TrashToPurgeView` | `TrashEmptied`, `ItemDeleted` | Items an empty-trash still owes a delete | EmptyTrashReactor |

The items read model is split into content (`ItemsView`) and state (`ItemStateView`) —
the same content-vs-location boundary as the events. At implementation these two join into
one `items` table (as the proposal specifies), and the enrichment events
(`LinkMetadataFetched`, `ThumbnailGenerated`) additionally fill the title/description and
thumbnail columns. A note stores its block list on the item row and each image in
`item_images`, ordered by position; the read model hydrates image metadata into the blocks
so clients can render a note without a second request. The Inbox is the set of captured
items whose state is still "inbox".

## Autonomous Components

| Component | Owned events | Owned commands | Consumes from |
|---|---|---|---|
| Identity | `SignedIn`, `SignedOut` | `SignIn`, `SignOut` | Google |
| Item | `TextCaptured`…`ItemDeleted`, `ItemTagged`, `LinkMetadataFetched`, `ThumbnailGenerated`, `NoteCaptured`, `ImageThumbnailGenerated` | capture / enrich / keep / tag / drop / restore / delete | — |
| Trash | `TrashEmptied` | `EmptyTrash` | — |
| Tag | `TagCreated`, `TagRenamed` | `CreateTag`, `RenameTag` | — |

Item and Trash share the `Item` context: `TrashEmptied` fans out to `DeleteItem` on the
`Item` aggregate.

## External Integrations

| Integration | Direction | Pattern | Events involved |
|---|---|---|---|
| Google OIDC | inbound | Translation | callback → `SignIn` → `SignedIn` |
| Target page fetch | outbound | Gateway | `LinkCaptured` → fetch → `RecordLinkMetadata` → `LinkMetadataFetched` |

## Message Processing Patterns

Aggregate (every command→event) · Projection (event→view) · Reactor (`LinkMetadataReactor`,
`ThumbnailReactor`, `EmptyTrashReactor`) · Translation (Google) · Gateway (link-metadata
fetch). No saga, process manager, policy, or scheduler.

## Implementation Slices

| # | Slice | Component | Events | Commands | Views |
|---|---|---|---|---|---|
| 1 | Sign In / Sign Out | Identity | `SignedIn`, `SignedOut` | `SignIn`, `SignOut` | — |
| 2 | Capture text | Item | `TextCaptured` | `CaptureText` | `ItemsView` |
| 3 | Capture link + metadata | Item | `LinkCaptured`, `LinkMetadataFetched` | `CaptureLink`, `RecordLinkMetadata` | `ItemsView`, `LinksToEnrichView` |
| 4 | Capture image + thumbnail | Item | `ImageCaptured`, `ThumbnailGenerated` | `CaptureImage`, `RecordThumbnail` | `ItemsView`, `ImagesToThumbnailView` |
| 5 | Capture file | Item | `FileCaptured` | `CaptureFile` | `ItemsView` |
| 6 | Keep + tag | Item | `ItemKept`, `ItemTagged` | `KeepItem`, `TagItem` | `ItemsView`, `ItemTagsView`, `ItemStateView` |
| 7 | Drop / restore / delete | Item | `ItemDropped`, `ItemRestored`, `ItemDeleted` | `DropItem`, `RestoreItem`, `DeleteItem` | `ItemStateView` |
| 8 | Empty trash | Trash | `TrashEmptied`, `ItemDeleted` | `EmptyTrash`, `DeleteItem` | `TrashToPurgeView` |
| 9 | Tags | Tag | `TagCreated`, `TagRenamed` | `CreateTag`, `RenameTag` | `TagsView` |
| 10 | Capture note with inline images | Item | `NoteCaptured`, `ImageThumbnailGenerated` | `CaptureNote`, `RecordImageThumbnail` | `ItemsView`, `ImagesToThumbnailView` |

## Decisions & Assumptions

- **Separate capture events, one `Item` aggregate** — the type is the event type; the
  location state machine is the only invariant and is uniform across types.
- **`ItemKept` (location) and `ItemTagged` (tags) are orthogonal** — one tag event,
  produced by keep and retag.
- **`TagNameTaken` is set-level** — enforced by a name index, not a per-tag stream.
- **`TrashNotEmpty`** — refuse to empty an empty trash; a mild UX rule, not in the stories.
- **Keep-without-tags** — left open; the stories say "can assign", not "must".
- **Restore target** — derived from history (inbox if never kept, library if kept); the
  stories flag this as an open question and the model assumes it.
- **Notes are one item with ordered blocks** — images are placed inline at the caret in the
  composer, not collected into a separate image item or a thumbnail strip. The document
  travels in `NoteCaptured`, so the note is one triage unit.
- **Images are uploaded with the note** — the composer keeps pasted images in memory and
  sends the document plus the files in one multipart request on Save. Limits are 10 images
  and 100 MB per note, with the existing 25 MB per-image limit.
