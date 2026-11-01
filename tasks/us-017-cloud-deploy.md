# US-017: Deploy to a cloud provider

## Decisions

- **The code change is the storage adapter.** Cloud deploy means the file bytes live in an
  S3-compatible bucket instead of local disk, so the API and `catchup` survive restarts and
  scale. The existing `storage.Storage` interface already hides Disk; this story adds S3
  behind it and selects the backend via config (`STORAGE_BACKEND=disk|s3`).
- **AWS SDK for Go v2**, talking to any S3-compatible endpoint (AWS S3, Cloudflare R2, MinIO)
  via `S3_ENDPOINT`. `S3_PATH_STYLE` opts into path-style addressing (MinIO needs it).
- **Managed Postgres needs no code** — the `DB` config already accepts an external host, so a
  cloud database is a matter of env, not a change.
- **Images publish to GHCR** on push to `main` (and tags), so a cloud host pulls the same
  images the self-hosted compose builds. The backend images are fully env-configured; the web
  image bakes the Google client ID as a build arg (unchanged from US-016).
- **The cloud compose drops `db`** (managed Postgres is external) and keeps `migrate` as a
  one-off that runs against the external database.

## Codebase patterns to follow

- `storage.NewDisk` is called in exactly two places (`cmd/catchup/main.go` and
  `internal/api/use_cases/register.go`); both switch to a `storage.New(ctx, cfg.Storage)`
  factory.
- `caarlos0/env` parses nested struct fields by their own `env:` tags, so a shared
  `storage.Config` embeds cleanly into `api.Config` and the catchup config.

## Tasks (dependency order)

### Phase 1 — S3 storage adapter
1. A shared `storage.Config` (backend, dir, and S3 fields) and a `New` factory that returns
   Disk or S3; an `S3` implementation of `Storage` using `PutObject`/`GetObject`.
   - `server/internal/storage/storage.go`, `disk.go`, `s3.go`, `config.go`.

### Phase 2 — wire it in
2. Switch both construction sites to the factory and embed `storage.Config` in each config.
   - `server/cmd/catchup/main.go`, `server/internal/api/config.go`,
     `server/internal/api/use_cases/register.go`.

### Phase 3 — tests
3. Unit tests: S3 `Save`/`Open` against a fake S3 client, and the factory's backend selection.
   - `server/internal/storage/s3_test.go`, `storage_test.go`.

### Phase 4 — publish + deploy
4. Publish `api`, `catchup`, and `web` to GHCR on push to `main`/tags.
   - `.github/workflows/ci.yml`.
5. A cloud compose (published images, external Postgres, S3 env) and its env template.
   - `server/docker-compose.cloud.yml`, `server/.env.cloud.example`.

### Phase 5 — docs
6. A cloud deploy doc: managed Postgres + S3 bucket + run the compose.
   - `docs/cloud-deploy.md`, linked from `README.md`.
