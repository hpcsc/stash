# US-016: Deploy self-hosted with docker compose

## Decisions

- **A separate production compose** (`server/docker-compose.production.yml`), distinct from the
  local-dev compose (`docker-compose.yml` + `docker-compose.local.yml`). It runs the whole
  self-hosted stack: `db`, `migrate`, `api`, `catchup`, and `web`.
- **Persistence via named volumes.** Postgres data lives in `db_data`; uploaded files and
  generated thumbnails live in `files`, mounted at `/data` (the `STORAGE_DIR`) in both `api`
  and `catchup`. Both survive a container restart, and the two processes share the files.
- **The web app is served same-origin by nginx.** nginx serves the built Svelte app and
  reverse-proxies `/auth`, `/items`, `/tags`, `/files`, `/trash` to `api`, plus `/ws` with the
  WebSocket upgrade headers. The app builds with an empty `VITE_API_BASE_URL`, so no hostname
  is baked into the image; the user reaches everything through one hostname (Tailscale).
- **Same-origin WebSocket.** When `VITE_API_BASE_URL` is empty, the client derives
  `ws://`/`wss://` from the page origin instead of rewriting the API base. Unset (dev) keeps the
  current behaviour.
- **Images build from source** (`build:`), not published. Publishing to a registry is a cloud
  concern and lands with US-017.
- **The API port stays exposed (3333)** so the Android app and the extension reach it directly,
  matching their existing `API_BASE_URL`. Tailscale is documented, not composed.

## Codebase patterns to follow

- The Go `DB` config reads `DB_HOST_PORT`, while the migrate entrypoint reads `DB_PORT`; the
  compose file sets each where it is consumed.
- `server/.env.local.example` is the template; the production compose gets
  `.env.production.example`, gitignored the same way.

## Tasks (dependency order)

### Phase 1 — same-origin web client
1. Derive the WebSocket URL from the page origin when the API base is empty.
   - `web/src/lib/config.ts`.

### Phase 2 — web image
2. A multi-stage web image (Node build → nginx) and an nginx config that serves the SPA and
   proxies the API and WebSocket.
   - `web/Dockerfile`, `web/nginx.conf`.

### Phase 3 — production compose
3. The compose file (db, migrate, api, catchup, web; `db_data` and `files` volumes) and its env
   template.
   - `server/docker-compose.production.yml`, `server/.env.production.example`.
4. Ignore the production env file.
   - `server/.gitignore`.

### Phase 4 — tasks + docs
5. `self-host:up` and `self-host:down` tasks.
   - `Taskfile.yml`.
6. A self-hosting doc: bring the stack up, set the env, point the phone at Tailscale.
   - `docs/self-hosting.md`, linked from `README.md`.
