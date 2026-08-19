variable "environment" {
  description = "RentStage environment name used in resource names and descriptions."
  type        = string

  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be staging or production."
  }
}

variable "environment_short" {
  description = "Short environment suffix used where Google service-account IDs are length constrained."
  type        = string

  validation {
    condition     = contains(["stg", "prod"], var.environment_short)
    error_message = "environment_short must be stg or prod."
  }
}

variable "project_id" {
  description = "Billing-enabled Google Cloud project dedicated to this RentStage environment."
  type        = string
}

variable "region" {
  description = "Region shared by Cloud Run, Artifact Registry, and Cloud SQL."
  type        = string
}

variable "artifact_repository" {
  description = "Docker Artifact Registry repository ID."
  type        = string
}

variable "deploy_service_account_email" {
  description = "Environment-specific GitHub deployment service account."
  type        = string
}

variable "database_name" {
  type = string
}

variable "database_user" {
  type = string
}

variable "database_password" {
  description = "PostgreSQL password supplied from the protected GitHub Environment."
  type        = string
  sensitive   = true
}

variable "database_tier" {
  description = "Cloud SQL machine tier selected for this environment."
  type        = string
}

variable "database_availability_type" {
  description = "ZONAL for cost-controlled environments or REGIONAL for high availability."
  type        = string

  validation {
    condition     = contains(["ZONAL", "REGIONAL"], var.database_availability_type)
    error_message = "database_availability_type must be ZONAL or REGIONAL."
  }
}

variable "database_disk_size" {
  description = "Initial Cloud SQL SSD size in GiB."
  type        = number

  validation {
    condition     = var.database_disk_size >= 10
    error_message = "database_disk_size must be at least 10 GiB."
  }
}

variable "backup_retention_count" {
  description = "Number of retained automated Cloud SQL backups."
  type        = number

  validation {
    condition     = var.backup_retention_count >= 7
    error_message = "backup_retention_count must retain at least seven backups."
  }
}

variable "deletion_protection" {
  description = "Protect the database at both Terraform and Cloud SQL levels."
  type        = bool
}

variable "additional_authorized_domains" {
  description = "Additional Identity Platform domains for this environment."
  type        = list(string)
}

variable "enable_meta_secret_containers" {
  description = "Create empty Secret Manager containers for the future Meta Cloud API adapter. Secret values are never placed in Terraform state."
  type        = bool
  default     = false
}

variable "enable_deploy_iam_bindings" {
  description = "Grant the environment deployment identity runtime deployment permissions. Keep false until that environment has an approved deployment workflow."
  type        = bool
  default     = false
}
