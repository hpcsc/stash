# Self-hosting

Run Stash on your own machine with `docker compose`, and reach it from your phone over
[Tailscale](https://tailscale.com) — no public exposure, no TLS to manage.

The stack is five containers: PostgreSQL, the migration runner, the API, the event
consumer (`catchup`), and nginx (which serves the web app and proxies the API's `/api/*`
routes). Data lives in two named volumes so it survives a container restart:

| Volume | Holds |
| --- | --- |
| `db_data` | The PostgreSQL event store and read models |
| `files` | Uploaded files and generated thumbnails |

## 1. One-time setup

```sh
cp server/.env.production.example server/.env.production
```

Set the values in `server/.env.production`:

- `SESSION_SECRET` — a long random string (e.g. `openssl rand -hex 32`).
- `GOOGLE_CLIENT_ID` — the Google OAuth **web** client ID from
  [docs/google-sign-in.md](google-sign-in.md). Add the Tailscale hostname from step 3 to its
  **authorised JavaScript origins**.
- `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD` — the database credentials (defaults are fine).

## 2. Start it

```sh
task self-host:up
```

This builds the images and starts the stack. The first build takes a few minutes; later
starts are fast. Logs: `docker compose -f server/docker-compose.production.yml logs -f`.

## 3. Reach it

- Web app: `http://<your-hostname>:8080`
- API (for the Android app and the browser extension): `http://<your-hostname>:3333`

Where `<your-hostname>` is the machine's LAN IP, or its Tailscale hostname
(`<machine>.<tailnet>.ts.net`) if you want access away from home.

For Tailscale, install it on the host machine and use its MagicDNS name. Point the Android
app at `http://<machine>.<tailnet>.ts.net:3333` and the browser extension's `apiBaseUrl` at
the same URL.

## 4. Stop it

```sh
task self-host:down
```

This stops the containers but keeps the volumes, so your data survives. To also delete the
data (irreversible):

```sh
docker compose --env-file server/.env.production -f server/docker-compose.production.yml down -v
```

## Upgrading

```sh
git pull
task self-host:up
```

The images rebuild with the new code, and the migration runner applies any new migrations
before the API and consumer start. `restart: unless-stopped` brings the stack back up after
a reboot.
