locals {
  api_runtime_service_account_id = "rentstage-api-stg"
  web_runtime_service_account_id = "rentstage-web-stg"
  sql_instance_name              = "rentstage-staging-postgres"
  required_services = toset([
    "apikeys.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "firebase.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "identitytoolkit.googleapis.com",
    "run.googleapis.com",
    "securetoken.googleapis.com",
    "secretmanager.googleapis.com",
    "serviceusage.googleapis.com",
    "sqladmin.googleapis.com",
    "sts.googleapis.com",
  ])
}

data "google_project" "current" {
  project_id = var.project_id
}

resource "google_project_service" "required" {
  for_each           = local.required_services
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

resource "google_artifact_registry_repository" "rentstage" {
  project       = var.project_id
  location      = var.region
  repository_id = var.artifact_repository
  description   = "RentStage staging container images"
  format        = "DOCKER"
  depends_on    = [google_project_service.required]
}

resource "google_service_account" "api_runtime" {
  project      = var.project_id
  account_id   = local.api_runtime_service_account_id
  display_name = "RentStage staging API runtime"
  depends_on   = [google_project_service.required]
}

resource "google_service_account" "web_runtime" {
  project      = var.project_id
  account_id   = local.web_runtime_service_account_id
  display_name = "RentStage staging web runtime"
  depends_on   = [google_project_service.required]
}

resource "google_sql_database_instance" "postgres" {
  project             = var.project_id
  name                = local.sql_instance_name
  region              = var.region
  database_version    = "POSTGRES_18"
  deletion_protection = var.deletion_protection

  settings {
    edition                     = "ENTERPRISE"
    tier                        = var.database_tier
    availability_type           = "ZONAL"
    disk_type                   = "PD_SSD"
    disk_size                   = 20
    disk_autoresize             = true
    deletion_protection_enabled = var.deletion_protection

    backup_configuration {
      enabled                        = true
      start_time                     = "03:00"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 7
        retention_unit   = "COUNT"
      }
    }

    ip_configuration {
      ipv4_enabled = true
    }

    maintenance_window {
      day          = 7
      hour         = 5
      update_track = "stable"
    }

    insights_config {
      query_insights_enabled  = true
      query_string_length     = 1024
      record_application_tags = false
      record_client_address   = false
    }
  }

  depends_on = [google_project_service.required]
}

resource "google_sql_database" "rentstage" {
  project  = var.project_id
  name     = var.database_name
  instance = google_sql_database_instance.postgres.name
}

resource "google_sql_user" "rentstage" {
  project  = var.project_id
  name     = var.database_user
  instance = google_sql_database_instance.postgres.name
  password = var.database_password
}

resource "random_id" "fingerprint_salt" {
  byte_length = 32
}

locals {
  database_url = format(
    "postgresql://%s:%s@/%s?host=%s&sslmode=disable",
    urlencode(var.database_user),
    urlencode(var.database_password),
    urlencode(var.database_name),
    urlencode("/cloudsql/${google_sql_database_instance.postgres.connection_name}"),
  )
}

resource "google_secret_manager_secret" "database_url" {
  project   = var.project_id
  secret_id = "rentstage-staging-database-url"
  replication {
    auto {}
  }
  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "database_url" {
  secret      = google_secret_manager_secret.database_url.id
  secret_data = local.database_url
}

resource "google_secret_manager_secret" "fingerprint_salt" {
  project   = var.project_id
  secret_id = "rentstage-staging-fingerprint-salt"
  replication {
    auto {}
  }
  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "fingerprint_salt" {
  secret      = google_secret_manager_secret.fingerprint_salt.id
  secret_data = random_id.fingerprint_salt.hex
}

resource "google_firebase_project" "rentstage" {
  provider   = google-beta
  project    = var.project_id
  depends_on = [google_project_service.required]

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_apikeys_key" "firebase" {
  project      = var.project_id
  name         = "rentstage-staging-firebase"
  display_name = "RentStage staging Firebase Web API key"

  restrictions {
    api_targets {
      service = "identitytoolkit.googleapis.com"
    }
    api_targets {
      service = "securetoken.googleapis.com"
    }
  }

  depends_on = [google_project_service.required]
}

resource "google_firebase_web_app" "rentstage" {
  provider        = google-beta
  project         = var.project_id
  display_name    = "RentStage Staging"
  api_key_id      = google_apikeys_key.firebase.uid
  deletion_policy = "ABANDON"
  depends_on      = [google_firebase_project.rentstage]
}

data "google_firebase_web_app_config" "rentstage" {
  provider   = google-beta
  project    = var.project_id
  web_app_id = google_firebase_web_app.rentstage.app_id
}

resource "google_identity_platform_config" "rentstage" {
  provider = google-beta
  project  = var.project_id

  sign_in {
    allow_duplicate_emails = false
    anonymous {
      enabled = false
    }
    email {
      enabled           = true
      password_required = true
    }
  }

  authorized_domains = distinct(concat([
    "localhost",
    "${var.project_id}.firebaseapp.com",
    "${var.project_id}.web.app",
  ], var.additional_authorized_domains))

  # The first Cloud Run URL does not exist during Terraform bootstrap. The
  # deployment workflow adds that exact hostname through the Identity Platform
  # Admin API after the web service is created. Keep Terraform from removing it
  # on later plans while retaining ownership of the rest of this configuration.
  lifecycle {
    ignore_changes = [authorized_domains]
  }

  depends_on = [google_firebase_project.rentstage, google_project_service.required]
}

resource "google_secret_manager_secret" "firebase_api_key" {
  project   = var.project_id
  secret_id = "rentstage-staging-firebase-api-key"
  replication {
    auto {}
  }
  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "firebase_api_key" {
  secret      = google_secret_manager_secret.firebase_api_key.id
  secret_data = data.google_firebase_web_app_config.rentstage.api_key
}

locals {
  api_project_roles = toset([
    "roles/cloudsql.client",
    "roles/firebaseauth.admin",
    "roles/logging.logWriter",
    "roles/serviceusage.serviceUsageConsumer",
  ])
  web_project_roles = toset([
    "roles/logging.logWriter",
  ])
  deploy_project_roles = toset([
    "roles/artifactregistry.writer",
    "roles/cloudsql.viewer",
    "roles/run.admin",
    "roles/identityplatform.admin",
    "roles/secretmanager.viewer",
    "roles/serviceusage.serviceUsageConsumer",
  ])
}

resource "google_project_iam_member" "api_runtime" {
  for_each = local.api_project_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_project_iam_member" "web_runtime" {
  for_each = local.web_project_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.web_runtime.email}"
}

resource "google_project_iam_member" "deploy" {
  for_each = local.deploy_project_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${var.deploy_service_account_email}"
}

resource "google_service_account_iam_member" "deploy_acts_as_api" {
  service_account_id = google_service_account.api_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${var.deploy_service_account_email}"
}

resource "google_service_account_iam_member" "deploy_acts_as_web" {
  service_account_id = google_service_account.web_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${var.deploy_service_account_email}"
}


resource "google_secret_manager_secret_iam_member" "api_database_url" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.database_url.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "api_fingerprint_salt" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.fingerprint_salt.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "deploy_firebase_api_key" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.firebase_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.deploy_service_account_email}"
}
