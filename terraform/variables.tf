variable "project_name" {
  description = "Prefix for resource names."
  type        = string
  default     = "stash"
}

# --- Google Cloud ---

variable "gcp_project_id" {
  description = "Google Cloud project ID."
  type        = string
}

variable "gcp_sa_key" {
  description = "Google Cloud service account key JSON (set from GOOGLE_CREDENTIALS in CI)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "gcp_region" {
  description = "Region for the VM and its network. The Always Free e2-micro is only free in us-west1, us-central1, and us-east1."
  type        = string
  default     = "us-central1"

  validation {
    condition     = contains(["us-west1", "us-central1", "us-east1"], var.gcp_region)
    error_message = "The Always Free e2-micro is only free in us-west1, us-central1, and us-east1."
  }
}

variable "gcp_zone" {
  description = "Zone for the VM, in the region above."
  type        = string
  default     = "us-central1-a"
}

variable "instance_type" {
  description = "Machine type. e2-micro is the Always Free one."
  type        = string
  default     = "e2-micro"
}

variable "boot_disk_size_gb" {
  description = "Boot disk size in GB. The free tier covers 30 GB of standard persistent disk."
  type        = number
  default     = 30
}

# --- Neon ---

variable "neon_org_id" {
  description = "Neon organisation ID (Account Settings → Organization settings)."
  type        = string
}

variable "neon_api_key" {
  description = "Neon API key (set from NEON_API_KEY in CI)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "neon_region_id" {
  description = "Neon region (e.g. aws-us-east-1, aws-ap-southeast-2)."
  type        = string
  default     = "aws-us-east-1"
}

# --- Cloudflare R2 ---

variable "cloudflare_account_id" {
  description = "Cloudflare account ID."
  type        = string
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token (set from CLOUDFLARE_API_TOKEN in CI)."
  type        = string
  sensitive   = true
  default     = ""
}

variable "r2_bucket_name" {
  description = "R2 bucket name for file storage."
  type        = string
  default     = "stash"
}
