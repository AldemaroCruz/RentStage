variable "project_id" {
  description = "Billing-enabled Google Cloud project dedicated to RentStage production."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id must be a valid Google Cloud project ID."
  }
}

variable "region" {
  description = "Production region shared by Cloud Run, Artifact Registry, and Cloud SQL."
  type        = string
  default     = "us-east1"
}

variable "artifact_repository" {
  description = "Production Docker Artifact Registry repository ID."
  type        = string
  default     = "rentstage"
}

variable "deploy_service_account_email" {
  description = "Production GitHub deployment service account created by bootstrap-gcp-production.sh."
  type        = string
}

variable "database_name" {
  type    = string
  default = "rentstage"
}

variable "database_user" {
  type    = string
  default = "rentstage"
}

variable "database_password" {
  description = "Production PostgreSQL password supplied from the GitHub production Environment."
  type        = string
  sensitive   = true

  validation {
    condition     = length(var.database_password) >= 24 && length(var.database_password) <= 64 && can(regex("^[A-Za-z0-9]+$", var.database_password))
    error_message = "database_password must be 24-64 alphanumeric characters so it can be represented safely in every deployment layer."
  }
}

variable "database_tier" {
  description = "Production Cloud SQL tier. Review the monthly estimate before the first apply."
  type        = string
  default     = "db-custom-1-3840"
}

variable "database_availability_type" {
  description = "ZONAL is the cost-controlled launch default. Change to REGIONAL when the availability SLA justifies the additional cost."
  type        = string
  default     = "ZONAL"

  validation {
    condition     = contains(["ZONAL", "REGIONAL"], var.database_availability_type)
    error_message = "database_availability_type must be ZONAL or REGIONAL."
  }
}

variable "database_disk_size" {
  description = "Initial production Cloud SQL SSD size in GiB."
  type        = number
  default     = 20

  validation {
    condition     = var.database_disk_size >= 20
    error_message = "Production database_disk_size must be at least 20 GiB."
  }
}

variable "backup_retention_count" {
  description = "Retained automated production backups."
  type        = number
  default     = 14

  validation {
    condition     = var.backup_retention_count >= 7
    error_message = "backup_retention_count must retain at least seven backups."
  }
}

variable "additional_authorized_domains" {
  description = "Production custom domains authorized by Identity Platform."
  type        = list(string)
  default     = []
}
