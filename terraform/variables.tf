variable "project_name" {
  description = "Prefix for resource names."
  type        = string
  default     = "stash"
}

# --- Oracle Cloud Infrastructure ---

variable "oci_tenancy_ocid" {
  description = "OCI tenancy OCID."
  type        = string
}

variable "oci_user_ocid" {
  description = "OCI user OCID."
  type        = string
}

variable "oci_fingerprint" {
  description = "Fingerprint of the OCI API key."
  type        = string
}

variable "oci_private_key" {
  description = "PEM content of the OCI API private key."
  type        = string
  sensitive   = true
}

variable "oci_region" {
  description = "OCI region (e.g. ap-sydney-1, us-ashburn-1)."
  type        = string
  default     = "ap-sydney-1"
}

variable "oci_compartment_ocid" {
  description = "OCI compartment to provision into (the root compartment OCID works)."
  type        = string
}

variable "instance_shape" {
  description = "OCI shape. Always Free: VM.Standard.E2.1.Micro (AMD) or VM.Standard.A1.Flex (ARM)."
  type        = string
  default     = "VM.Standard.E2.1.Micro"
}

variable "instance_ocpus" {
  description = "OCPUs for a flex shape."
  type        = number
  default     = 1
}

variable "instance_memory_in_gbs" {
  description = "Memory (GB) for a flex shape. A1.Flex minimum is 6."
  type        = number
  default     = 6
}

# --- Neon ---

variable "neon_org_id" {
  description = "Neon organisation ID (Account Settings → Organization settings)."
  type        = string
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

variable "r2_bucket_name" {
  description = "R2 bucket name for file storage."
  type        = string
  default     = "stash"
}
