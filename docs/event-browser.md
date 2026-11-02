# Event browser

A read-only browser for the event store, for debugging. It runs on your machine as a small
local web server and reads the store through the Stash API.

The main web app does not include it. Only an operator who holds the debug token can open it.

## How it works

The CLI serves a local page and forwards every `/api/events…` request to the backend API
with the debug token in the `Authorization` header. The token stays in the CLI process, so
the browser never holds it and it never appears in a URL.

```
browser  ──►  http://127.0.0.1:7777  ──►  Stash API  ──►  Postgres
              (CLI, adds the token)
```

The backend needs `DEBUG_EVENTS_TOKEN` set. When it is empty the API refuses the token scope,
and the browser shows only the streams of a signed-in user.

## Set the token on the backend

The token is a deployment secret. Generate one and put it where the other secrets live:

```sh
openssl rand -hex 32
```

Set `DEBUG_EVENTS_TOKEN` in `server/.env.production` (self-host) or `server/.env.cloud` (cloud),
then redeploy. The API refuses a configured token shorter than 32 characters.

The token is not printed by the API, the CLI, or the page. Keep it in your deployment secret
store, and give it only to the people who debug the store.

## Run the browser

Build the UI into the CLI, then start it:

```sh
task event-browser:build

./server/bin/eventbrowser \
  --url https://stash.example.com \
  --token "$STASH_DEBUG_TOKEN"
```

Then open <http://127.0.0.1:7777>.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--url` | — | Base URL of the Stash API. Required. |
| `--token` | `$STASH_DEBUG_TOKEN` | Debug token. Required. |
| `--addr` | `127.0.0.1:7777` | Where the local UI listens. |

The address defaults to loopback, so the page is not reachable from another machine. Pass a
different `--addr` only if you know why you need it: the token scope reads every user's streams.

## What you can see

- **Events** — the store in commit order. Filter by type, stream prefix, correlation id, or a
  time range, and follow new events with **Live**.
- **Streams** — a tree of every stream in the scope. Open a stream to read it in version order.
- **Checkpoints** — each consumer's position and how far it lags the head of the store.

An event opens in a side panel with its raw payload, its metadata, and the decoded struct. An
event the build cannot decode still opens: the panel shows the parse error instead of failing.

Every read is a `SELECT`. The browser cannot rewrite, replay, or delete an event.

## Develop the page

```sh
task event-browser:build      # build once, so the CLI has the current UI
./server/bin/eventbrowser --url http://127.0.0.1:3333 --token "$STASH_DEBUG_TOKEN"

task web:event-browser:dev    # Vite with hot reload, proxying /api to the CLI on 7777
```
