#!/usr/bin/env bash
set -euo pipefail

# Automates as much of the one-time GCP + Neon + R2 setup as practical.
# Read the full rationale at docs/deploy.md.
#
# What it does:
#   1. Checks that the required CLIs are installed.
#   2. Creates (or selects) a Google Cloud project, enables Compute Engine,
#      creates a service account for Terraform, and downloads its key.
#   3. Prompts for the Neon org ID + API key.
#   4. Prompts for the Cloudflare account ID, R2 API token, and R2 S3 creds.
#   5. Prompts for the HCP Terraform org name, creates the "stash" workspace,
#      and replaces the placeholder in terraform/backend.tf.
#   6. Prompts for the Google OAuth web client ID.
#   7. Generates a SESSION_SECRET.
#   8. Pushes everything to GitHub as repository secrets and variables.
#   9. Prints a checklist of what remains for a human.

readonly REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
readonly TF_BACKEND="$REPO_ROOT/terraform/backend.tf"
readonly SA_NAME="stash-deploy"
readonly SA_ROLE="roles/compute.admin"
readonly REQUIRED_CLIS="gcloud gh curl openssl"

# ---- helpers ----

ask() {
  local prompt="$1"
  local var_name="$2"
  local val
  read -r -p "$prompt: " val
  printf -v "$var_name" '%s' "$val"
}

ask_secret() {
  local prompt="$1"
  local var_name="$2"
  local val
  echo "$prompt (the value will not echo):"
  read -r -s val
  echo
  printf -v "$var_name" '%s' "$val"
}

set_gh_secret() { gh secret set "$1" -b"$2" --repo "${GITHUB_REPO:-}"; }
set_gh_var()   { gh variable set "$1" -b"$2" --repo "${GITHUB_REPO:-}"; }

# ---- pre-flight ----

check_clis() {
  local missing=()
  for c in $REQUIRED_CLIS; do
    command -v "$c" >/dev/null 2>&1 || missing+=("$c")
  done
  if command -v terraform >/dev/null 2>&1; then :; else missing+=(terraform); fi
  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Missing CLIs: ${missing[*]}"
    echo "Install them and re-run."
    exit 1
  fi
}

check_gh_auth() {
  echo "  Checking GitHub CLI auth (enter 'gh auth login' if this hangs)..."
  if ! gh auth status >/dev/null 2>&1; then
    echo "Not authenticated with GitHub CLI. Run 'gh auth login' first."
    exit 1
  fi
  GITHUB_REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)"
  if [[ -z "$GITHUB_REPO" ]]; then
    echo "Could not determine the GitHub repository. Are you inside the repo?"
    exit 1
  fi
  echo "  GitHub repo: $GITHUB_REPO"
}

check_gcloud_auth() {
  if ! gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null | grep -q .; then
    echo "Not authenticated with gcloud. Run 'gcloud auth login' first."
    exit 1
  fi
}

# ---- GCP ----

setup_gcp() {
  echo
  echo "=== Google Cloud ==="
  echo

  local billing_account
  billing_account="$(gcloud beta billing accounts list --format='value(ACCOUNT_ID)' --limit=1 2>/dev/null || true)"

  if [[ -z "$billing_account" ]]; then
    echo "No billing account found. You need one (the free tier covers the VM,"
    echo "but Google requires a billing account on the project)."
    echo "Set one up at https://console.cloud.google.com/billing and re-run."
    exit 1
  fi

  echo "Billing account: $billing_account"
  echo "  (this can take a moment — gcloud is looking up your billing account)"
  echo

  echo "Do you want to:"
  echo "  1) create a new Google Cloud project"
  echo "  2) use an existing one"
  read -r -p "Choose (1/2) [1]: " choice
  choice="${choice:-1}"

  local project_id
  if [[ "$choice" == "2" ]]; then
    ask "    Existing project ID" project_id
    gcloud config set project "$project_id" 2>/dev/null || {
      echo "Failed to set project '$project_id'. Does it exist?"
      exit 1
    }
    if gcloud beta billing projects describe "$project_id" --format='value(billingEnabled)' 2>/dev/null | grep -q True; then :; else
      echo "    Linking billing account $billing_account to project $project_id..."
      gcloud beta billing projects link "$project_id" --billing-account "$billing_account"
    fi
  else
    local project_name
    ask "    New project name (e.g. stash)" project_name
    project_name="${project_name:-stash}"
    project_id="${project_name}-$(tr -dc 'a-z0-9' < /dev/urandom | head -c6)"
    echo "    Creating project: $project_id"
    gcloud projects create "$project_id" --name="$project_name" 2>/dev/null || {
      # retry with a different ID if it already exists
      project_id="${project_name}-$(date +%s)"
      echo "    Retrying with ID: $project_id"
      gcloud projects create "$project_id" --name="$project_name"
    }
    gcloud config set project "$project_id"
    echo "    Linking billing account..."
    gcloud beta billing projects link "$project_id" --billing-account "$billing_account"
  fi

  echo "    Enabling Compute Engine API (takes 30-60 seconds for the billing account to become active)..."
  local waited=0
  while ! gcloud services enable compute.googleapis.com --project "$project_id" 2>/dev/null; do
    if [[ $waited -ge 120 ]]; then
      echo
      echo "    Billing is still not active. This means the billing account linked to"
      echo "    '$project_id' is closed or needs a payment method verified."
      echo "    Open this page to activate it:"
      echo "      https://console.cloud.google.com/billing"
      echo "    Then re-run this script."
      echo
      echo "    To check which billing account is linked:"
      echo "      gcloud beta billing projects describe $project_id"
      exit 1
    fi
    echo -n "."
    sleep 10
    waited=$((waited + 10))
  done
  echo
  echo "    Compute Engine API enabled."

  echo "    Creating service account '$SA_NAME'..."
  if gcloud iam service-accounts describe "$SA_NAME@$project_id.iam.gserviceaccount.com" \
       --project "$project_id" >/dev/null 2>&1; then
    echo "    Service account already exists."
  else
    gcloud iam service-accounts create "$SA_NAME" --project "$project_id"
    gcloud projects add-iam-policy-binding "$project_id" \
      --member "serviceAccount:$SA_NAME@$project_id.iam.gserviceaccount.com" \
      --role "$SA_ROLE"
    # Allow the service account to be used by other resources (VM, GH Actions).
    gcloud projects add-iam-policy-binding "$project_id" \
      --member "serviceAccount:$SA_NAME@$project_id.iam.gserviceaccount.com" \
      --role roles/iam.serviceAccountUser 2>/dev/null || true
  fi

  local key_file="$REPO_ROOT/key.json"
  if [[ -f "$key_file" ]]; then
    echo "    Key file '$key_file' already exists. Delete it to re-create."
  else
    echo "    Creating service account key..."
    gcloud iam service-accounts keys create "$key_file" \
      --iam-account "$SA_NAME@$project_id.iam.gserviceaccount.com" \
      --project "$project_id"
    echo "    Key written to $key_file (do not commit it)."
  fi

  GCP_PROJECT_ID="$project_id"
}

# ---- Neon ----

setup_neon() {
  echo
  echo "=== Neon (neon.tech) ==="
  echo
  echo "These steps are manual — no CLI to create an org or API key."
  echo
  echo "  1. Go to https://console.neon.tech and sign up (or sign in)."
  echo "  2. Account Settings → Organization settings → copy your Organization ID."
  echo "  3. Account Settings → API Keys → create a key and copy it."
  echo

  ask "    Neon organization ID" NEON_ORG_ID
  ask_secret "    Neon API key" NEON_API_KEY
}

# ---- Cloudflare ----

setup_cloudflare() {
  echo
  echo "=== Cloudflare R2 ==="
  echo
  echo "  1. Go to https://dash.cloudflare.com and sign in."
  echo "  2. Copy your Account ID from the dashboard home page."
  echo "  3. Create an R2 API token (Profile → API Tokens → Create Token →"
  echo "     R2 read/write scope). Copy the token."
  echo "  4. R2 → Manage R2 API Tokens → Create API Token (the S3-compatible"
  echo "     credentials). Copy the Access Key ID and Secret Access Key."
  echo

  ask "    Cloudflare account ID" CLOUDFLARE_ACCOUNT_ID
  ask_secret "    R2 API token" CLOUDFLARE_API_TOKEN
  ask_secret "    R2 S3 access key ID" R2_ACCESS_KEY_ID
  ask_secret "    R2 S3 secret access key" R2_SECRET_ACCESS_KEY
}

# ---- HCP Terraform ----

setup_terraform_cloud() {
  echo
  echo "=== HCP Terraform (app.terraform.io) ==="
  echo
  echo "  1. Go to https://app.terraform.io and sign up (or sign in)."
  echo "  2. Create an organization (or use an existing one)."
  echo "  3. User Settings → Tokens → create an API token and copy it."
  echo
  echo "  The script will create the 'stash' workspace for you."
  echo

  ask "    Terraform Cloud organization name" TF_ORG

  echo "    Replacing placeholder in $TF_BACKEND..."
  if grep -q "YOUR-TERRAFORM-CLOUD-ORG" "$TF_BACKEND"; then
    sed -i '' "s/YOUR-TERRAFORM-CLOUD-ORG/$TF_ORG/" "$TF_BACKEND"
    echo "    Done."
  else
    echo "    Placeholder already replaced (or absent) — skipping."
  fi

  ask_secret "    Terraform Cloud API token" TF_TOKEN_APP_TERRAFORM_IO

  echo
  echo "    Creating workspace 'stash' in $TF_ORG..."
  local resp
  resp="$(curl -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $TF_TOKEN_APP_TERRAFORM_IO" \
    -H "Content-Type: application/vnd.api+json" \
    -d '{"data":{"type":"workspaces","attributes":{"name":"stash","resource-count":0}}}' \
    "https://app.terraform.io/api/v2/organizations/$TF_ORG/workspaces" 2>&1 || true)"
  local status
  status="$(echo "$resp" | tail -1)"
  local body
  body="$(echo "$resp" | sed '$d')"

  if [[ "$status" == "201" ]]; then
    echo "    Workspace 'stash' created."
  elif [[ "$status" == "422" ]]; then
    echo "    Workspace 'stash' already exists."
  else
    echo "    Could not create workspace (HTTP $status). Response:"
    echo "    $body"
    echo
    echo "    You can create it manually at:"
    echo "    https://app.terraform.io/app/$TF_ORG/workspaces/new"
    echo "    Or check that your token has permission to create workspaces."
  fi

  echo
  echo "    IMPORTANT: After the workspace exists, grant the token's team"
  echo "    at least 'write' access to it. Without this, terraform apply in"
  echo "    CI will fail with a 404."
  echo "    Open this page and check Team Access:"
  echo "    https://app.terraform.io/app/$TF_ORG/workspaces/stash/settings/team-access"
}

# ---- Google OAuth ----

setup_oauth() {
  echo
  echo "=== Google OAuth ==="
  echo
  echo "  In the Google Cloud Console (https://console.cloud.google.com/apis/credentials):"
  echo "    APIs & Services → Credentials → your Web Application client"
  echo "  Under Authorised JavaScript origins, add:"
  echo "    http://localhost:8080"
  echo "    http://127.0.0.1:8080"
  echo
  echo "  (After the first deploy you'll also add https://<your-vm-ip>.)"
  echo "  If you don't have a client yet, see scripts/google-auth-setup.sh."
  echo

  ask "    Google OAuth web client ID" GOOGLE_CLIENT_ID
}

# ---- push to GitHub ----

push_to_github() {
  echo
  echo "=== Setting GitHub secrets and variables ==="
  echo

  local key_json
  key_json="$(cat "$REPO_ROOT/key.json")"

  set_gh_secret GCP_SA_KEY "$key_json"
  set_gh_var    GCP_PROJECT_ID "$GCP_PROJECT_ID"

  set_gh_secret NEON_API_KEY "$NEON_API_KEY"
  set_gh_var   NEON_ORG_ID "$NEON_ORG_ID"

  set_gh_secret CLOUDFLARE_API_TOKEN "$CLOUDFLARE_API_TOKEN"
  set_gh_var   CLOUDFLARE_ACCOUNT_ID "$CLOUDFLARE_ACCOUNT_ID"
  set_gh_secret R2_ACCESS_KEY_ID "$R2_ACCESS_KEY_ID"
  set_gh_secret R2_SECRET_ACCESS_KEY "$R2_SECRET_ACCESS_KEY"

  set_gh_secret TF_TOKEN_APP_TERRAFORM_IO "$TF_TOKEN_APP_TERRAFORM_IO"

  # SESSION_SECRET is generated, not prompted.
  local session_secret
  session_secret="$(openssl rand -hex 32)"
  set_gh_secret SESSION_SECRET "$session_secret"

  set_gh_secret GOOGLE_CLIENT_ID "$GOOGLE_CLIENT_ID"

  echo "  All secrets and variables set."

  # Clean up the local key file — it is now in the secret store.
  rm -f "$REPO_ROOT/key.json"
}

# ---- summary ----

print_summary() {
  echo
  echo "====================="
  echo "  Setup complete"
  echo "====================="
  echo
  echo "Automated:"
  echo "  ✓ GCP project '$GCP_PROJECT_ID'"
  echo "  ✓ Service account '$SA_NAME' with Compute Admin"
  echo "  ✓ Compute Engine API enabled"
  echo "  ✓ Terraform backend updated ($TF_BACKEND)"
  echo "  ✓ HCP Terraform workspace 'stash'"
  echo "  ✓ SESSION_SECRET generated"
  echo "  ✓ All GitHub secrets and variables set"
  echo
  echo "Manual steps still needed:"
  echo "  • Review terraform/backend.tf — confirm the organisation name is correct."
  echo "  • After the first deploy: add the VM IP to the OAuth client's origins."
  echo "  • The first terraform apply (in CI or locally) provisions the real resources."
  echo "    Run it locally first: cd terraform && terraform init && terraform plan"
  echo
  echo "To deploy, push to main or trigger the CI workflow from the Acions tab."
}

# ---- main -----

main() {
  echo "Stash — one-time GCP provisioner"
  echo

  check_clis
  check_gh_auth
  check_gcloud_auth

  setup_gcp
  setup_neon
  setup_cloudflare
  setup_terraform_cloud
  setup_oauth
  push_to_github
  print_summary
}

main "$@"