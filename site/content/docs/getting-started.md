---
title: Getting started
weight: 1
---

Stash is a self-hosted capture inbox. You run one Go service backed by PostgreSQL, then
capture into it from the web app, the Chrome or Firefox extension, or the Android app.

## Requirements

- [mise](https://mise.jdx.dev/) for Go, `task`, and the other tools
- Docker, `jq`, and `tmuxp`

```sh
mise install
```

## Run it locally

```sh
cp server/.env.local.example server/.env.local   # once
task run
```

`task run` starts Postgres on port 45432, runs the migrations, and opens a tmux window with
the `api` and `catchup` processes. The API listens on `:3333`.

`server/.env.local` holds the local configuration: the database settings and the
`SESSION_SECRET` and `GOOGLE_CLIENT_ID` placeholders. The API starts with the placeholders,
but signing in needs a real Google OAuth client ID — see
[Google sign-in](/docs/google-sign-in/).

## Capture something

The item routes use the session token from sign-in:

```sh
TOKEN=$(curl -s -X POST http://127.0.0.1:3333/auth/signin \
  -d '{"provider":"google","idToken":"<google-id-token>"}' | jq -r .data.token)

curl -s -X POST http://127.0.0.1:3333/items \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"body":"remember this","sourceApp":"web"}' | jq

curl -s http://127.0.0.1:3333/items -H "Authorization: Bearer $TOKEN" | jq
```

## Test

```sh
task test:unit
task test:integration   # starts a separate Postgres on port 45433
```
