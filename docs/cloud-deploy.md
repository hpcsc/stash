# Cloud deployment

Run Stash on a cloud provider without running a server at home. The stack is the same as
self-hosting, but the file bytes live in an S3-compatible bucket and the database is a
managed Postgres, so nothing survives on the VM's disk and the API and consumer can restart
(or scale) without losing anything.

Two images — `ghcr.io/hpcsc/stash-api` and `ghcr.io/hpcsc/stash-catchup` — are published to
GitHub Container Registry on every push to `main` (see `.github/workflows/publish.yml`).
The web app is built at deploy time because it bakes the Google client ID in.

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
builds and starts the web app, and starts the API and consumer. Logs:

```sh
docker compose --env-file server/.env.cloud -f server/docker-compose.cloud.yml logs -f
```

## 4. Reach it

- Web app: `https://<your-domain-or-ip>:8080`
- API (for the Android app and the browser extension): `https://<your-domain-or-ip>:3333`

Put the web app behind TLS (a reverse proxy, or the provider's load balancer) so Google
sign-in works over HTTPS.

## 5. Stop it

```sh
task cloud:down
```

## Publishing images

Images are published automatically by the `Publish images` workflow. To push them by hand:

```sh
docker build -t ghcr.io/hpcsc/stash-api:main   -f server/Dockerfile.api     server
docker build -t ghcr.io/hpcsc/stash-catchup:main -f server/Dockerfile.catchup server
docker push ghcr.io/hpcsc/stash-api:main
docker push ghcr.io/hpcsc/stash-catchup:main
```

The compose file points at the `:main` tag; pin it to a `v*` tag for a reproducible deploy.
