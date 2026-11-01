# Deployment to GCP + Neon + R2

Provision the whole cloud stack with Terraform — a free-tier Google Cloud VM, a Neon
Postgres, and a Cloudflare R2 bucket — and deploy the app to the VM from GitHub Actions. One
workflow builds and pushes the backend images, then deploys.

```
push to main
  └─► CI (this workflow)
        ├─► test      (shellcheck, unit, integration, vulnerabilities)
        ├─► publish   (needs test; build + push api/catchup to GHCR; api bundles the web app)
        └─► deploy    (needs publish)
              ├─► terraform apply  (GCP VM + Neon project + R2 bucket)
              └─► SSH to the VM → docker compose up  (pulls api/catchup, runs migrations)
```

The API serves the built web app and its own `/api/*` routes from one container, so there is
no separate web/nginx container. Cloudflare Tunnel forwards the API port (3333) to the edge.

The VM is an `e2-micro` in `us-central1`, which the Google Cloud free tier covers: the machine
and a 30 GB standard disk. A Cloudflare Tunnel provides TLS and avoids a static external IP
charge; the VM gets a free ephemeral IP for outbound traffic.

## One-time setup

### Quick start

Most of the setup is one script:

```sh
scripts/provision-gcp.sh
```

It asks for each value as it goes, creates the GCP project and service account, updates the
Terraform backend, creates the HCP Terraform workspace, generates a session secret, and sets
all the GitHub secrets and variables at the end.

The sections below explain each step so you know what the script does, and they document the
manual dashboard steps that cannot be scripted (Neon sign-up, Cloudflare tokens, OAuth client).

### 1. Google Cloud

The script handles this end-to-end:

- Creates (or selects) a project and links a billing account.
- Enables the Compute Engine API.
- Creates the `stash-deploy` service account with **Compute Admin**.
- Downloads the service account key to `key.json` (and deletes it after pushing it to
  GitHub — the key is never committed).

To run just the GCP steps interactively:
```sh
gcloud services enable compute.googleapis.com --project <project-id>
gcloud iam service-accounts create stash-deploy --project <project-id>
gcloud projects add-iam-policy-binding <project-id> \
  --member serviceAccount:stash-deploy@<project-id>.iam.gserviceaccount.com \
  --role roles/compute.admin
gcloud iam service-accounts keys create key.json \
  --iam-account stash-deploy@<project-id>.iam.gserviceaccount.com \
  --project <project-id>
```

The free tier only covers `us-west1`, `us-central1`, and `us-east1`; Terraform rejects any
other region.

### 2. Neon

1. Sign up at neon.tech. Note your **organization ID** (Account Settings → Organization).
2. Create an **API key** (Account Settings → API Keys).

### 3. Cloudflare R2

1. Note your **account ID** (dashboard home).
2. Create an **API token** with R2 read/write scope (for Terraform).
3. Separately, create the S3 credentials the app uses: **R2 → Manage R2 API Tokens →
   Create API token**. Note the **access key ID** and **secret access key**. Terraform can
   create the bucket but not these S3 tokens, so they are a manual step.

### 4. HCP Terraform (state)

The script replaces the placeholder in `terraform/backend.tf` and creates the workspace.
Before running it you need:

1. An account + organization at app.terraform.io.
2. An **API token** (User Settings → Tokens).

When prompted, give the script your organization name and API token. It creates a workspace
named `stash`.

Without the script:
```sh
# 1. Edit terraform/backend.tf: replace YOUR-TERRAFORM-CLOUD-ORG with your org.
# 2. Create the stash workspace via the CLI:
export TF_TOKEN_app_terraform_io="<your-api-token>"
curl -X POST \
  -H "Authorization: Bearer $TF_TOKEN_app_terraform_io" \
  -H "Content-Type: application/vnd.api+json" \
  -d '{"data":{"type":"workspaces","attributes":{"name":"stash"}}}' \
  "https://app.terraform.io/api/v2/organizations/<org>/workspaces"
```

### 5. Google OAuth

Add the Cloudflare Tunnel origin to the web OAuth client's **authorised JavaScript origins**.
The tunnel URL appears in the deploy log (e.g. `https://stash-xxx.trycloudflare.com`).
If you switch to a custom domain later, add that origin too.

The script prompts for the OAuth client ID itself.

### 6. GitHub secrets and variables

The script sets every value below via `gh secret set` / `gh variable set`. It also generates
a `SESSION_SECRET` (`openssl rand -hex 32`) and deletes the local `key.json` afterward.

| Name | Kind | Value |
| --- | --- | --- |
| `GCP_SA_KEY` | secret | Service account key JSON (the script pushes it from `key.json`) |
| `GCP_PROJECT_ID` | variable | Google Cloud project ID |
| `NEON_API_KEY` | secret | Neon API key |
| `NEON_ORG_ID` | variable | Neon organization ID |
| `CLOUDFLARE_API_TOKEN` | secret | Cloudflare API token |
| `CLOUDFLARE_ACCOUNT_ID` | variable | Cloudflare account ID |
| `R2_ACCESS_KEY_ID` | secret | R2 S3 access key ID |
| `R2_SECRET_ACCESS_KEY` | secret | R2 S3 secret access key |
| `SESSION_SECRET` | secret | Generated by the script |
| `GOOGLE_CLIENT_ID` | secret | Google OAuth web client ID |
| `TF_TOKEN_APP_TERRAFORM_IO` | secret | Terraform Cloud API token |

Defaults that usually need no override: the region (`us-central1`), the zone
(`us-central1-a`), the machine type (`e2-micro`), the Neon region (`aws-us-east-1`), the R2
bucket name (`stash`). Change them by adding matching repository variables and passing them
as `TF_VAR_*`.

## Deploy

Push to `main`, or run the workflow manually from the Actions tab. The `CI` workflow:

1. Runs the shellcheck, unit, integration, and vulnerability checks. Nothing is published or
   deployed unless they pass.
2. Builds the `api` and `catchup` images and pushes them to GHCR. The `api` image bundles
   the built web app (the Google client ID is baked in as a build arg).
3. `terraform apply` — provisions (or updates) the VM, the Neon project, and the R2 bucket.
4. Writes `server/.env.cloud` on the VM from the Terraform outputs and the app secrets.
5. Clones (or pulls) the repo on the VM and runs `docker compose up` — pulls the published
   `api`/`catchup` images, runs the migrations, and starts everything. The `api` container
   serves both the web app and the API.

A `v*` tag pushes images but skips the deploy. The first deploy waits for the VM's first-boot
script to install Docker and add a swap file.

Reach the app at the tunnel URL printed in the deploy log (a `*.trycloudflare.com` hostname).
Add that origin to your Google OAuth client's authorised JavaScript origins before signing in.

The tunnel is a free Cloudflare Tunnel running as a systemd service on the VM. It forwards
the API port (3333) to the Cloudflare edge with automatic TLS, so the app has HTTPS with no
certificate setup and no static external IP.

## Caveats

- **Terraform is untested against real accounts.** The configuration is `terraform validate`-
  clean, but the first `apply` needs your GCP/Neon/Cloudflare credentials and may need small
  adjustments. Run it locally first with `cd terraform && terraform init && terraform plan`.
- **R2 S3 credentials are manual.** Terraform creates the bucket; the S3 access key + secret
  come from the dashboard and are GitHub secrets.
- **The VM is small.** `e2-micro` gives 1 GB of RAM and a quarter of a core. Nothing is built
  on it — the `api` image is built in CI and pulled — so the small machine is plenty. The
  Terraform startup script still adds a 2 GB swap file as a safety net.
- **The external IP is free.** The VM gets an ephemeral IP for outbound traffic (the tunnel
  needs it to connect to Cloudflare). Terraform no longer provisions a static IP.
- **State lives in HCP Terraform.** If you'd rather not use it, remove `terraform/backend.tf`
  and store the state elsewhere (e.g. a GCS bucket), otherwise deploys are not idempotent.
- **The repo is cloned publicly on the VM.** If the repository is private, the clone step
  needs a deploy key instead.
- **TLS is provided by Cloudflare Tunnel.** The API serves plain HTTP on `:3333`; `cloudflared`
  adds TLS between the browser and the Cloudflare edge. No certificate management needed.
