# Deployment to GCP + Neon + R2

Provision the whole cloud stack with Terraform — a free-tier Google Cloud VM, a Neon
Postgres, and a Cloudflare R2 bucket — and deploy the app to the VM from GitHub Actions. One
workflow builds and pushes the backend images, then deploys.

```
push to main
  └─► CI (this workflow)
        ├─► test      (shellcheck, unit, integration, vulnerabilities)
        ├─► publish   (needs test; build + push api/catchup to GHCR)
        └─► deploy    (needs publish)
              ├─► terraform apply  (GCP VM + Neon project + R2 bucket)
              └─► SSH to the VM → docker compose up  (pulls api/catchup, builds web)
```

The VM is an `e2-micro` in `us-central1`, which the Google Cloud free tier covers: the machine,
a 30 GB standard disk, and 1 GB of egress per month. The **external IPv4 address is not part
of the free tier** — it costs about $3.65/month.

## One-time setup

### 1. Google Cloud

1. Create a project and enable the Compute Engine API:
   ```sh
   gcloud services enable compute.googleapis.com --project <project-id>
   ```
2. Create a service account for Terraform, with **Compute Admin** and **Service Account User**:
   ```sh
   gcloud iam service-accounts create stash-deploy --project <project-id>
   gcloud projects add-iam-policy-binding <project-id> \
     --member serviceAccount:stash-deploy@<project-id>.iam.gserviceaccount.com \
     --role roles/compute.admin
   gcloud iam service-accounts keys create key.json \
     --iam-account stash-deploy@<project-id>.iam.gserviceaccount.com \
     --project <project-id>
   ```
3. Note the **project ID** (`key.json` goes into a GitHub secret).

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

The deploy stores Terraform state in HCP Terraform's free tier so it survives across
ephemeral GitHub runners.

1. Create an account + organization at app.terraform.io, then a workspace named `stash`.
2. In `terraform/backend.tf`, replace `YOUR-TERRAFORM-CLOUD-ORG` with your organization.
3. Create a Terraform Cloud **API token**.

### 5. Google OAuth

Add the VM's public origin to the web OAuth client's **authorised JavaScript origins**
(`https://<vm-ip-or-domain>`). You'll know the IP after the first deploy; `terraform output
instance_public_ip` prints it.

### 6. GitHub secrets and variables

| Name | Kind | Value |
| --- | --- | --- |
| `GCP_SA_KEY` | secret | Service account key JSON (from step 1) |
| `GCP_PROJECT_ID` | variable | Google Cloud project ID |
| `NEON_API_KEY` | secret | Neon API key |
| `NEON_ORG_ID` | variable | Neon organization ID |
| `CLOUDFLARE_API_TOKEN` | secret | Cloudflare API token |
| `CLOUDFLARE_ACCOUNT_ID` | variable | Cloudflare account ID |
| `R2_ACCESS_KEY_ID` | secret | R2 S3 access key ID |
| `R2_SECRET_ACCESS_KEY` | secret | R2 S3 secret access key |
| `SESSION_SECRET` | secret | `openssl rand -hex 32` |
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
2. Builds the `api` and `catchup` images and pushes them to GHCR.
3. `terraform apply` — provisions (or updates) the VM, the Neon project, and the R2 bucket.
4. Writes `server/.env.cloud` on the VM from the Terraform outputs and the app secrets.
5. Clones (or pulls) the repo on the VM and runs `docker compose up` — pulls the published
   `api`/`catchup` images, builds the web app, runs the migrations, and starts everything.

A `v*` tag pushes images but skips the deploy. The first deploy waits for the VM's first-boot
script to install Docker and add a swap file.

Reach the app at `http://<vm-ip>:8080`.

## Caveats

- **Terraform is untested against real accounts.** The configuration is `terraform validate`-
  clean, but the first `apply` needs your GCP/Neon/Cloudflare credentials and may need small
  adjustments. Run it locally first with `cd terraform && terraform init && terraform plan`.
- **R2 S3 credentials are manual.** Terraform creates the bucket; the S3 access key + secret
  come from the dashboard and are GitHub secrets.
- **The VM is small.** `e2-micro` gives 1 GB of RAM and a quarter of a core, so building the
  web image on it is slow (minutes, not seconds). Terraform adds a 2 GB swap file so the build
  is not killed. If that is too slow, publish the web image to GHCR as well and have the cloud
  compose pull it.
- **The external IP is billed.** The address is static so it survives restarts and keeps the
  OAuth origin valid; at $0.005/hour it is about $3.65/month. Going IPv6-only or reaching the
  VM through a Cloudflare Tunnel would remove the charge.
- **State lives in HCP Terraform.** If you'd rather not use it, remove `terraform/backend.tf`
  and store the state elsewhere (e.g. a GCS bucket), otherwise deploys are not idempotent.
- **The repo is cloned publicly on the VM.** If the repository is private, the clone step
  needs a deploy key instead.
- **No TLS on the VM.** The compose file serves plain HTTP on `:8080`; put a TLS terminator in
  front of it before pointing Google sign-in at it.
