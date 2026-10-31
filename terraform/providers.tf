provider "oci" {
  tenancy_ocid = var.oci_tenancy_ocid
  user_ocid    = var.oci_user_ocid
  fingerprint  = var.oci_fingerprint
  private_key  = var.oci_private_key
  region       = var.oci_region
}

provider "neon" {
  # Reads NEON_API_KEY from the environment.
}

provider "cloudflare" {
  # Reads CLOUDFLARE_API_TOKEN from the environment.
}
