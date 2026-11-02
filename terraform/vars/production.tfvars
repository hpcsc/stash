# Production values for the GCP + Neon + R2 + Tunnel stack.
# Secrets (API keys, tokens) still come from GitHub secrets → TF_VAR_* env vars and
# must never live in this file.  The values below are identifiers, not credentials.

domain        = "saphix.app"
app_subdomain = "stash"

gcp_project_id        = "stash-513410"
neon_org_id           = "org-fragrant-heart-42810323"
cloudflare_account_id = "ceb312854aca938fd572ef74be79bd2d"

# Override defaults only when they differ from variables.tf.
# project_name     = "stash"
# gcp_region       = "us-central1"
# gcp_zone         = "us-central1-a"
# instance_type    = "e2-micro"
# boot_disk_size_gb = 30
# neon_region_id   = "aws-us-east-1"
# r2_bucket_name   = "stash"