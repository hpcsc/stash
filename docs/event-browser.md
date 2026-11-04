# Event browser

A read-only browser for the event store, for debugging. It runs on your machine as a small
local web server and reads the store through the Stash API.

The main web app does not include it. The API decides what the browser can read from the
session the CLI holds: your own streams, or every user's streams when your Google subject id
is listed in `EVENT_BROWSER_USER_IDS`.

## How it works

The CLI serves a local page with no login of its own. The page asks you to sign in with
Google and sends the Google ID token to the CLI. The CLI exchanges it at
`POST /api/auth/signin`, keeps the resulting session token, and puts it in the
`Authorization` header of every `/api/events…` request it forwards. The session token stays
in the CLI process, so the browser never holds it and it never appears in a URL.

```
browser  ──►  http://127.0.0.1:7777  ──►  Stash API  ──►  Postgres
              (CLI, holds the session token)
```

The API resolves every request to one of two scopes:

| Credential | Scope |
| --- | --- |
| Session token of a user listed in `EVENT_BROWSER_USER_IDS` | every user's streams |
| Session token of any other signed-in user | that user's streams only |
| Nothing, or an unknown or expired token | 401 |

## Sign in

Open <http://127.0.0.1:7777> and use the **Sign in with Google** button. The CLI needs one
thing from the Google console: `http://127.0.0.1:7777` must be an authorised JavaScript
origin for the Web OAuth client, next to the web app's origins
([google-sign-in.md](google-sign-in.md)).

The CLI stores the session token in the user config directory with mode `0600`
(`~/.config/stash` on Linux, `~/Library/Application Support` on macOS). It keeps one file per
backend, named `stash/event-browser-session-<hash>`, so a local token and a production token
do not overwrite each other. It loads the token on the next start, so you sign in about once a
month. A session lasts 30 days and there is no refresh: when it expires the API answers 401,
the CLI drops the token, and the page asks you to sign in again.

Signing out clears that file and also calls `POST /api/auth/signout`, so the store records
`stash.signed-out` the same way a web sign-out does. A token from `--token` is left alone, and
the page does not offer to sign it out.

The session token is a normal account credential, as capable as signing in to the web app.
Treat it like a password; the CLI keeps it out of the page and out of a URL, but it is still
readable by anything running as your user.

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

Build the UI into the CLI, then start it:

```sh
task event-browser:build
./server/bin/eventbrowser --url https://stash.example.com
```

Then open <http://127.0.0.1:7777> and sign in.

To read production, run `task event-browser:run:production`, which points the CLI at
`https://stash.saphix.app`. Override the host with `STASH_EVENT_BROWSER_PRODUCTION_URL`.

For local development, run `task event-browser:run`, which loads `server/.env.local`.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--url` | — | Base URL of the Stash API. Required. |
| `--token` | `$STASH_SESSION_TOKEN` | Optional session token for headless use. Without it, the page signs in. |
| `--addr` | `127.0.0.1:7777` | Where the local UI listens. |

The address defaults to loopback, so the page is not reachable from another machine. Pass a
different `--addr` only if you know why you need it: the all-stream scope reads every user's
streams. Anyone who can reach the port can use the session the CLI holds.

### Headless use

For a script, or a machine where you cannot open a browser, pass a session token from the
web app's devtools (`localStorage.session_token`):

```sh
STASH_SESSION_TOKEN=<your token> ./server/bin/eventbrowser --url https://stash.example.com
```

Prefer the environment variable over `--token` so the token stays out of the process list. A
token supplied this way is fixed for the life of the process: the page cannot sign out or
replace it.

## What you can see

- **Events** — the store in commit order. Filter by type, stream prefix, correlation id, or a
  time range, and follow new events with **Live**.
- **Streams** — a tree of every stream in the scope. Open a stream to read it in version order.
- **Checkpoints** — each consumer, whether it is a projection or a reaction, the event types it
  reads, how far it lags those types, and when it last progressed. A lagging consumer shows the
  next event it will process. A projection also opens to show its read model, so its state can
  be checked against the events that produced it.

An event opens in a side panel with its raw payload, its metadata, and the decoded struct. An
event the build cannot decode still opens: the panel shows the parse error instead of failing.

Every read is a `SELECT`. The browser cannot rewrite, replay, or delete an event.

## Develop the page

```sh
task event-browser:build      # build once, so the CLI has the current UI
./server/bin/eventbrowser --url http://127.0.0.1:3333

task web:event-browser:dev    # Vite with hot reload, proxying /api and /session to 7777
```
