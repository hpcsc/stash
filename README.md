# Stash

A self-hosted capture inbox: save text, links, images, and files from any app on a phone or
desktop, then sort them into tags or discard them. Event-sourced Go, with the event store and
its projections both in PostgreSQL.

## Repository

- `server/` — the Go service: the event store, the HTTP API, and the projections. Everything
  the `US-*` stories build.
- `app/` — the Flutter client for Android and Web. Everything the `APP-*` stories build.
- `docs/`, `user-stories/`, `tasks/` — shared: the event model, the stories, and the plans.

## Status

Google sign-in and text capture are implemented end to end. The event model for the whole
feature lives in [docs/event-model-stash.md](docs/event-model-stash.md), with the source in
`event-model-stash.emod` and the diagram in `event-model-stash.drawio`. The user stories are
in `user-stories/stash.md`.

## Architecture

- `server/cmd/api` — the HTTP API. Dispatches commands and reads projections. Sign-in verifies a
  Google ID token, records `SignedIn`, and returns a session token; the auth middleware
  validates that token and carries the subject.
- `server/cmd/catchup` — projections and reactions. No consumers yet; the items projection arrives
  with the next story.
- `server/internal/domain` — the event store, the aggregate/command/event base, projections,
  reactions, checkpoints, and subscriptions. See
  [docs/event-subscription.md](docs/event-subscription.md).
- `server/internal/use_cases/identity` — the sign-in domain: an `Account` aggregate with `SignedIn`
  and `SignedOut`.

## Requirements

- [mise](https://mise.jdx.dev/) for Go, `task`, `migrate`, `gotestsum`, `shellcheck`, and
  `govulncheck`
- Docker, `jq`, and `tmuxp`

```sh
mise install
```

## Run locally

```sh
task run
```

Starts Postgres on port 45432, runs the migrations, and opens a tmux window with the `api`
and `catchup` processes.

The API reads `SESSION_SECRET` and `GOOGLE_CLIENT_ID` from the environment. `task api:run`
sets development placeholders, so the API starts, but signing in needs a real Google OAuth
client ID.

## Sign in

Get a Google ID token whose audience is your client ID — the
[OAuth 2.0 Playground](https://developers.google.com/oauthplayground) is the quickest way —
then:

```sh
TOKEN=$(curl -s -X POST http://127.0.0.1:3333/auth/signin \
  -d '{"provider":"google","idToken":"<google-id-token>"}' | jq -r .data.token)
```

Send the token as a bearer on the protected routes:

```sh
curl -s -X POST http://127.0.0.1:3333/auth/signout -H "Authorization: Bearer $TOKEN" | jq
```

## Capture text

The item routes use the session token from sign-in:

```sh
curl -s -X POST http://127.0.0.1:3333/items \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"body":"remember this","sourceApp":"web"}' | jq

curl -s http://127.0.0.1:3333/items -H "Authorization: Bearer $TOKEN" | jq

curl -s http://127.0.0.1:3333/items/<item-id> -H "Authorization: Bearer $TOKEN" | jq
```

## Test

```sh
task test:unit
task test:integration   # starts a separate Postgres on port 45433
```

Unit tests carry the `unit` build tag; integration tests carry the `integration` tag and use
`server/.env.integration`.

## Layout

| Path | Contents |
| --- | --- |
| `server/cmd/api` | The HTTP API. |
| `server/cmd/catchup` | The projections and reactions. |
| `server/internal/domain` | Events, aggregates, commands, and their bus. |
| `server/internal/domain/event/stream` | The write path: append to a stream, read a stream. |
| `server/internal/domain/event/subscription` | The read loop and the listener. |
| `server/internal/domain/consumer` | The loop for every consumer. |
| `server/internal/domain/projection`, `server/internal/domain/reaction` | The two kinds of consumer. |
| `server/internal/domain/checkpoint/store` | Checkpoints in Postgres. |
| `server/internal/use_cases/<domain>` | The aggregate, events, commands, and repository of a domain. |
| `server/internal/use_cases/<domain>/<use case>` | One use case: its command handler, and its projection or reaction. |
| `server/internal/use_cases/register.go` | The registration of event types and command handlers. |
| `server/internal/api` | The HTTP server, routes, middleware, and handlers. |
| `server/migrations` | The schema. |
| `docs` | The event model and the event-subscription design. |

## Add an Event

1. Add a struct for the payload to the `event` package of the domain, for example `SignedIn`
   in `server/internal/use_cases/identity/event`.
2. Give the struct an `EventType` method that returns the name of the event type.
3. Register the struct in the `RegisterEvents` function of the domain, for example in
   `server/internal/use_cases/identity/events.go`.

The store saves the name of the event type with each event. When the store reads the event,
the registry finds the struct from that name. You can rename the struct, but do not change the
name of the event type after the store has an event of that type.

## Change an Event

Each saved event keeps the schema version of its payload struct. A registration with no schema
version has version 1.

| Change | What to do |
| --- | --- |
| Add a field that an old build can ignore | Change the struct only. |
| Add a field that an old build must not ignore | Change the struct. Increase the schema version, and pass `nil` as the upcast. |
| Rename or remove a field, or change what a field means | Change the struct. Increase the schema version, and give an upcast. |

```go
reg.Register[event.SignedIn](registry.WithSchemaVersion(2, upcastSignedIn))
```

The upcast gets the schema version and the data of an older event, and returns the data in the
current shape. The registry calls it before it parses the data.

When an old build reads an event with a newer schema version, the registry returns
`registry.ErrNewerSchemaVersion`, and the read loop stops. Therefore an old `catchup` stops
during a deploy, and does not write wrong data to a read model. The `catchup` of the new build
reads the event.

## Add a Command

1. Add a method to the aggregate. The method checks the rule and calls the private `record`
   method of the aggregate with the new event, for example
   `domain.NewEvent(&event.SignedIn{...})`. `Apply` changes the state from that event.
2. Add a payload type for the command, for example `SignIn` in
   `server/internal/use_cases/identity/command`.
3. Add a package for the use case, for example `server/internal/use_cases/identity/signin`. Give it a
   handler type with a `Handle(ctx, domain.Command[SignIn]) error` method. The handler loads
   the aggregate from its repository, calls the method, and saves.
4. Give the package a `Register` function that calls `bus.Register[SignIn](handler{...})`, and
   call it from `server/internal/use_cases/register.go`.

Send the command with `bus.Dispatch(ctx, domain.NewCommand(SignIn{...}))`. The bus finds the
handler from the payload type, and calls it again on a concurrency conflict.

## Add a Consumer

- **A projection:** implement `projection.Interface`. Write to the read model through the
  `pgx.Tx` that `Handle` gets. Add the table in a migration. Add
  `projection.NewProjector(...)` to the consumers in `server/cmd/catchup/main.go`.
- **A process manager:** a projection that keeps its state, and appends the requests it
  decides as events in the same transaction, plus a reaction with
  `reaction.StartAtBeginning()` that sends each request.
- **A reaction:** implement `reaction.Interface`, or use `external.NewHttpReaction`. Make the
  effect safe to repeat. A message to another service is a contract, not an event, as
  [docs/event-subscription.md](docs/event-subscription.md) tells. A reaction that sends a
  command takes the command ID from `dispatch.CommandIDFor`.

The name of each consumer is the key of its checkpoint. Do not rename a consumer after it runs
in production: a projection with a new name rebuilds from the beginning, and a reaction with a
new name starts again at the end of the store.
