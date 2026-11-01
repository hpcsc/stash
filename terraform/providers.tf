provider "google" {
  credentials = var.gcp_sa_key != "" ? var.gcp_sa_key : null
  project     = var.gcp_project_id
  region      = var.gcp_region
  zone        = var.gcp_zone
}

provider "neon" {
  api_key = var.neon_api_key
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}
