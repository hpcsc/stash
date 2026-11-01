provider "google" {
  # Reads GOOGLE_CREDENTIALS (service account key JSON) from the environment.
  project = var.gcp_project_id
  region  = var.gcp_region
  zone    = var.gcp_zone
}

provider "neon" {
  # Reads NEON_API_KEY from the environment.
}

provider "cloudflare" {
  # Reads CLOUDFLARE_API_TOKEN from the environment.
}
