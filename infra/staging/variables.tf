variable "project_id" {
  description = "Billing-enabled Google Cloud project used only for RentStage staging."
  type        = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id must be a valid Google Cloud project ID."
  }
}

variable "region" {
  description = "Region shared by Cloud Run, Artifact Registry, and Cloud SQL."
  type        = string
  default     = "us-east1"
}

variable "artifact_repository" {
  description = "Docker Artifact Registry repository ID."
  type        = string
  default     = "rentstage"
}

variable "deploy_service_account_email" {
  description = "GitHub deployment service account created by bootstrap-gcp-staging.sh."
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
  description = "Staging PostgreSQL password supplied from the GitHub staging Environment."
  type        = string
  sensitive   = true
  validation {
    condition     = length(var.database_password) >= 24 && length(var.database_password) <= 64 && can(regex("^[A-Za-z0-9]+$", var.database_password))
    error_message = "database_password must be 24-64 alphanumeric characters so it can be represented safely in every deployment layer."
  }
}

variable "database_tier" {
  type        = string
  default     = "db-f1-micro"
  description = "Low-cost staging tier; increase it before load testing."
}

variable "deletion_protection" {
  type        = bool
  default     = true
  description = "Protect the staging database from accidental Terraform deletion."
}

variable "additional_authorized_domains" {
  type        = list(string)
  default     = []
  description = "Optional custom staging domains for Identity Platform."
}
