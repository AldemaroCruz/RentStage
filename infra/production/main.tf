module "platform" {
  source = "../modules/rentstage-platform"

  environment                   = "production"
  environment_short             = "prod"
  project_id                    = var.project_id
  region                        = var.region
  artifact_repository           = var.artifact_repository
  deploy_service_account_email  = var.deploy_service_account_email
  database_name                 = var.database_name
  database_user                 = var.database_user
  database_password             = var.database_password
  database_tier                 = var.database_tier
  database_availability_type    = var.database_availability_type
  database_disk_size            = var.database_disk_size
  backup_retention_count        = var.backup_retention_count
  deletion_protection           = true
  additional_authorized_domains = var.additional_authorized_domains
  enable_meta_secret_containers = true
  enable_deploy_iam_bindings    = false
  use_minimal_firebase_role     = true
}
