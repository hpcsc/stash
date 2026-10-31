resource "neon_project" "main" {
  name           = var.project_name
  region_id      = var.neon_region_id
  org_id         = var.neon_org_id
  pg_version     = 17
  store_password = "yes"

  # Free-tier accounts cap point-in-time restore history at 6 hours.
  history_retention_seconds = 21600

  branch {
    database_name = "stash"
    role_name     = "stash"
  }
}
