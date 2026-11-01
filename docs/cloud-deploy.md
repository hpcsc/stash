# Cloud deployment

Run Stash on a cloud provider without running a server at home. The stack is the same as
self-hosting, but the file bytes live in an S3-compatible bucket and the database is a
managed Postgres, so nothing survives on the VM's disk and the API and consumer can restart
(or scale) without losing anything.

Two images — `ghcr.io/hpcsc/stash-api` and `ghcr.io/hpcsc/stash-catchup` — are published to
GitHub Container Registry on every push to `main` (see `.github/workflows/ci.yml`, the
`publish` job). The `api` image bundles the built web app (the Google client ID is baked in as
a build arg), so no web build happens at deploy time.

## Cheapest options (provider cheat sheet)

Nothing here is AWS-specific — the storage and database are chosen by env. The cheapest
paths, roughly in order:

| Need | Cheapest option | Cost |
| --- | --- | --- |
| Run everything at home | Your own machine + Tailscale | $0 |
| Run everything in the cloud | One free/cheap VM (Google Cloud Always Free `e2-micro`, Hetzner) | $0–4/mo |
| Managed Postgres | Neon or Supabase free tier | $0 |
| S3-compatible files | Cloudflare R2 free tier (no egress fees), Backblaze B2 | $0 |

### Option A — one free VM, everything on it

The **self-hosted** compose (`server/docker-compose.production.yml`) runs Postgres and stores
files on the VM's own disk, so there are no per-service bills. On Google Cloud's Always Free
tier the VM itself costs nothing — the external IPv4 address is billed at about $3.65/month.
Any cheap VPS works the same way.

1. Launch an **Always Free** `e2-micro` VM (in `us-west1`, `us-central1`, or `us-east1`) with
   Ubuntu, and open ports 8080 and 3333. It has 1 GB of RAM, so add a swap file if you build
   anything on it.
2. Install Docker and the Compose plugin:
   ```sh
   curl -fsSL https://get.docker.com | sh
   ```
3. Clone Stash and configure it:
   ```sh
   git clone https://github.com/hpcsc/stash.git && cd stash
   cp server/.env.production.example server/.env.production
   # set SESSION_SECRET and GOOGLE_CLIENT_ID
   ```
4. Start it (no `task` or `mise` needed):
   ```sh
   docker compose --env-file server/.env.production -f server/docker-compose.production.yml up -d --build
   ```
5. Put the web app behind TLS (a domain + Caddy or Let's Encrypt, or Tailscale) so Google
   sign-in works over HTTPS. Point the Android app and the extension at the API.

### Option B — Neon + R2 free tier (managed)

The **cloud** compose (`server/docker-compose.cloud.yml`) with free-tier managed services.
Copy `server/.env.cloud.example` to `server/.env.cloud` and fill in:

1. **Neon** — create a project and copy its connection string
   (`postgresql://user:pass@ep-…us-east-1.aws.neon.tech/neondb`). Split it into `DB_HOST`,
   `DB_PORT` (5432), `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD`. Keep `DB_SSLMODE=require` —
   Neon accepts only TLS connections.
2. **Cloudflare R2** — create a bucket and an S3 API token, then set:
   ```sh
   S3_BUCKET=<your-bucket>
   S3_REGION=auto
   S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
   S3_ACCESS_KEY_ID=<r2 access key id>
   S3_SECRET_ACCESS_KEY=<r2 secret key>
   S3_PATH_STYLE=true
   ```
3. Start it: `task cloud:up` (or
   `docker compose --env-file server/.env.cloud -f server/docker-compose.cloud.yml up -d`).

The published images are pulled from GHCR, so no build happens on the VM.

## 1. Provision the cloud resources

- **A VM** — any provider (a VPS, EC2, a Fly Machine, …). It needs Docker and outbound
  network access to the database and the bucket.
- **Managed Postgres** — Neon, RDS, Supabase, or similar. Note the host, port, database name,
  and credentials. The `migrate` one-off container applies the schema on first start.
- **An S3-compatible bucket** — AWS S3, Cloudflare R2, MinIO, or similar. Note the bucket
  name, region, and (for non-AWS) the endpoint. Give the VM credentials to the bucket:
  either an IAM role on the host (leave the keys empty) or an access key + secret key.

## 2. Configure

```sh
cp server/.env.cloud.example server/.env.cloud
```

Fill in `server/.env.cloud` with the database and bucket values above, plus:

- `SESSION_SECRET` — a long random string (e.g. `openssl rand -hex 32`).
- `GOOGLE_CLIENT_ID` — the Google OAuth **web** client ID (see
  [docs/google-sign-in.md](google-sign-in.md)). Add the public origin of the web app to its
  **authorised JavaScript origins**.

For MinIO, set `S3_PATH_STYLE=true`; for AWS S3 leave `S3_ENDPOINT` empty.

## 3. Start it

```sh
task cloud:up
```

This pulls the published backend images, runs the migrations against the managed Postgres,
and starts the API (which serves both the web app and the `/api/*` routes) and the consumer.
Logs:

```sh
docker compose --env-file server/.env.cloud -f server/docker-compose.cloud.yml logs -f
```

## 4. Reach it

The API container serves everything on one port: the web app on `/` and the API under `/api`.

- Web app: `https://<your-domain-or-ip>:3333`
- API (for the Android app and the browser extension): `https://<your-domain-or-ip>:3333/api`

Put the web app behind TLS (a reverse proxy, or the provider's load balancer) so Google
sign-in works over HTTPS.

## 5. Stop it

```sh
task cloud:down
```

## Publishing images

Images are published automatically by the `publish` job of the `CI` workflow. To push them
by hand:

```sh
docker build -t ghcr.io/hpcsc/stash-api:main   -f server/Dockerfile.api .
docker build -t ghcr.io/hpcsc/stash-catchup:main -f server/Dockerfile.catchup server
docker push ghcr.io/hpcsc/stash-api:main
docker push ghcr.io/hpcsc/stash-catchup:main
```

The compose file points at the `:main` tag; pin it to a `v*` tag for a reproducible deploy.
