# Event browser

A read-only browser for the event store, for debugging. It runs on your machine as a small
local web server and reads the store through the Stash API.

The main web app does not include it. The API decides what the browser can read from your
Stash session: your own streams, or every user's streams when your Google subject id is
listed in `EVENT_BROWSER_USER_IDS`.

## How it works

The CLI serves a local page and forwards every `/api/events…` request to the backend API
with your session token in the `Authorization` header. The token stays in the CLI process,
so the browser never holds it and it never appears in a URL.

```
browser  ──►  http://127.0.0.1:7777  ──►  Stash API  ──►  Postgres
              (CLI, adds the session token)
```

The API resolves every request to one of two scopes:

| Credential | Scope |
| --- | --- |
| Session token of a user listed in `EVENT_BROWSER_USER_IDS` | every user's streams |
| Session token of any other signed-in user | that user's streams only |
| Nothing, or an unknown or expired token | 401 |

## Get a session token

Sign in to the web app, then read the token from the browser devtools:

```js
localStorage.session_token
```

The same token carries the subject id the API checks against the list:

```js
JSON.parse(atob(localStorage.session_token.split('.')[1])).sub
```

A session lasts 30 days and there is no refresh. When it expires the API answers 401; sign in
again for a new token. It also grants normal access to the account, so treat it like a
password: do not paste it into a shared shell or a ticket.

## Let a user see every stream

By default a session reads only its own streams. To grant the all-stream scope, put the
user's Google subject id in `EVENT_BROWSER_USER_IDS` on the API:

```sh
EVENT_BROWSER_USER_IDS=113456789012345678901,104567890123456789012
```

The subject is the `sub` claim of the session token, which is also the user id in the store.
The API reads the list at startup, so a change needs a redeploy. The list is checked on every
request: removing a subject takes the scope away without invalidating that person's session.

## Run the browser

Build the UI into the CLI, then start it with your session token:

```sh
task event-browser:build

STASH_SESSION_TOKEN=<your token> \
  ./server/bin/eventbrowser --url https://stash.example.com
```

Then open <http://127.0.0.1:7777>.

For local development, put the token in `server/.env.local` as `STASH_SESSION_TOKEN` and run
`task event-browser:run`, which loads that file.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--url` | — | Base URL of the Stash API. Required. |
| `--token` | `$STASH_SESSION_TOKEN` | Session token. Required. |
| `--addr` | `127.0.0.1:7777` | Where the local UI listens. |

The address defaults to loopback, so the page is not reachable from another machine. Pass a
different `--addr` only if you know why you need it: the all-stream scope reads every user's
streams. The local page has no login of its own, so anyone who can reach the port can read
what the token can.

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
./server/bin/eventbrowser --url http://127.0.0.1:3333

task web:event-browser:dev    # Vite with hot reload, proxying /api to the CLI on 7777
```

The CLI reads the session token from `STASH_SESSION_TOKEN` when `--token` is not given.
