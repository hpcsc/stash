# Terraform state lives in HCP Terraform (free tier). Replace the organisation
# with your own, and create a workspace named "stash" there. The deploy workflow
# authenticates with TF_TOKEN_app_terraform_io.
#
# To run locally without Terraform Cloud, comment out the cloud block; the state
# then stays on disk (not suitable for GitHub Actions, which uses an ephemeral
# runner).
terraform {
  cloud {
    organization = "davidnguyen"

    workspaces {
      name = "stash"
    }
  }
}
