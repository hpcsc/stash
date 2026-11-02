# Event browser

A read-only browser for the event store, for debugging. It runs on your machine as a small
local web server and reads the store through the Stash API.

The main web app does not include it. It reads every user's streams, so the API grants that
scope only to a debug token or to a signed-in user on the event browser list. Without either, a
signed-in user sees only their own streams.

## How it works

The CLI serves a local page and forwards every `/api/events…` request to the backend API
with the token in the `Authorization` header. The token stays in the CLI process, so the
browser never holds it and it never appears in a URL.

```
browser  ──►  http://127.0.0.1:7777  ──►  Stash API  ──►  Postgres
              (CLI, adds the token)
```

## Access

The API resolves every request to one of two scopes:

| Credential | Scope |
| --- | --- |
| The `DEBUG_EVENTS_TOKEN` value | every user's streams |
| Session token of a user in `EVENT_BROWSER_USER_IDS` | every user's streams |
| Session token of any other signed-in user | that user's streams only |
| Nothing, or an unknown or expired token | 401 |

Both settings are optional. With both empty, the browser still works, but only for the
streams of the user whose session token it is given.

## Set the token on the backend

The token is a deployment secret. Generate one and put it where the other secrets live:

```sh
openssl rand -hex 32
```

Set `DEBUG_EVENTS_TOKEN` in `server/.env.production` (self-host) or `server/.env.cloud` (cloud),
then redeploy. The API refuses a configured token shorter than 32 characters.

The token is not printed by the API, the CLI, or the page. Keep it in your deployment secret
store, and give it only to the people who debug the store.

## Let signed-in users see every stream

Instead of sharing one secret, list the users who may see every stream:

```sh
EVENT_BROWSER_USER_IDS=113456789012345678901,104567890123456789012
```

Each entry is the Google subject id of a user, which is also the user id in the store. The
session token whose subject matches gets the all-stream scope; every other session keeps the
user scope. The list lives in the same env file as the other secrets, so changing it needs a
redeploy.

To read your own subject, sign in to the web app and run this in the browser devtools:

```js
JSON.parse(atob(localStorage.session_token.split('.')[1])).sub
```

Signing in again gives a fresh session; there is no refresh. A session lasts 30 days, and the
API answers 401 once it expires.

A session token also grants normal access to the account, so treat it like a password: do not
paste it into a shared shell or a ticket. To revoke access, remove the subject from the list
and redeploy.

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
| `--token` | `$STASH_DEBUG_TOKEN` | Debug token or session token. Required. |
| `--addr` | `127.0.0.1:7777` | Where the local UI listens. |

The address defaults to loopback, so the page is not reachable from another machine. Pass a
different `--addr` only if you know why you need it: the all-stream scope reads every user's
streams.

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
