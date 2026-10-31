# Deployment to Oracle + Neon + R2

Provision the whole cloud stack with Terraform — an Oracle VM, a Neon Postgres, and a
Cloudflare R2 bucket — and deploy the app to the VM from GitHub Actions. Every push to
`main` that publishes images also deploys.

```
push to main
  └─► Publish images (GHCR)
        └─► Deploy (this workflow)
              ├─► terraform apply  (Oracle VM + Neon project + R2 bucket)
              └─► SSH to the VM → docker compose up  (pulls api/catchup, builds web)
```

## One-time setup

### 1. Oracle Cloud

1. Create an Oracle Cloud account (a card is required even for Always Free) and note the
   **tenancy OCID** and the root **compartment OCID**.
2. Under **Identity → Users → your user → API Keys**, add an API key and download the
   **private key**. Note the **fingerprint** and your **user OCID**.

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
| `OCI_TENANCY_OCID` | secret | Tenancy OCID |
| `OCI_USER_OCID` | secret | User OCID |
| `OCI_FINGERPRINT` | secret | API key fingerprint |
| `OCI_PRIVATE_KEY` | secret | API private key (PEM) |
| `OCI_COMPARTMENT_OCID` | secret | Compartment OCID |
| `NEON_API_KEY` | secret | Neon API key |
| `NEON_ORG_ID` | variable | Neon organization ID |
| `CLOUDFLARE_API_TOKEN` | secret | Cloudflare API token |
| `CLOUDFLARE_ACCOUNT_ID` | variable | Cloudflare account ID |
| `R2_ACCESS_KEY_ID` | secret | R2 S3 access key ID |
| `R2_SECRET_ACCESS_KEY` | secret | R2 S3 secret access key |
| `SESSION_SECRET` | secret | `openssl rand -hex 32` |
| `GOOGLE_CLIENT_ID` | secret | Google OAuth web client ID |
| `TF_TOKEN_APP_TERRAFORM_IO` | secret | Terraform Cloud API token |

Defaults that usually need no override: `OCI_REGION` (ap-sydney-1), the instance shape
(AMD micro), `NEON_REGION_ID` (aws-us-east-1), the R2 bucket name (`stash`). Change them by
adding matching repository variables and passing them as `TF_VAR_*`.

## Deploy

Push to `main`, or run the workflow manually from the Actions tab. The `Deploy` workflow:

1. `terraform apply` — provisions (or updates) the VM, the Neon project, and the R2 bucket.
2. Writes `server/.env.cloud` on the VM from the Terraform outputs and the app secrets.
3. Clones (or pulls) the repo on the VM and runs `docker compose up` — pulls the published
   `api`/`catchup` images, builds the web app, runs the migrations, and starts everything.

Reach the app at `http://<vm-ip>:8080`.

## Caveats

- **Terraform is untested against real accounts.** The configuration is `terraform validate`-
  clean, but the first `apply` needs your OCI/Neon/Cloudflare credentials and will likely
  need small adjustments (free-tier shape availability, region, image). Run it locally first
  with `cd terraform && terraform init && terraform plan`.
- **R2 S3 credentials are manual.** Terraform creates the bucket; the S3 access key + secret
  come from the dashboard and are GitHub secrets.
- **OCI Always Free** ARM (A1.Flex) shapes are often out of capacity; the default AMD
  `VM.Standard.E2.1.Micro` is the safer choice.
- **State lives in HCP Terraform.** If you'd rather not use it, remove `terraform/backend.tf`
  and store the state elsewhere (e.g. an OCI Object Storage bucket), otherwise deploys are
  not idempotent.
- **The repo is cloned publicly on the VM.** If the repository is private, the clone step
  needs a deploy key instead.
