locals {
  api_runtime_service_account_id = "rentstage-api-${var.environment_short}"
  web_runtime_service_account_id = "rentstage-web-${var.environment_short}"
  sql_instance_name              = "rentstage-${var.environment}-postgres"
  resource_prefix                = "rentstage-${var.environment}"

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

  meta_secret_names = toset([
    "meta-access-token",
    "meta-app-id",
    "meta-app-secret",
    "meta-phone-number-id",
    "meta-waba-id",
    "meta-webhook-verify-token",
  ])
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
  description   = "RentStage ${var.environment} container images"
  format        = "DOCKER"
  depends_on    = [google_project_service.required]
}

resource "google_service_account" "api_runtime" {
  project      = var.project_id
  account_id   = local.api_runtime_service_account_id
  display_name = "RentStage ${var.environment} API runtime"
  depends_on   = [google_project_service.required]
}

resource "google_service_account" "web_runtime" {
  project      = var.project_id
  account_id   = local.web_runtime_service_account_id
  display_name = "RentStage ${var.environment} web runtime"
  depends_on   = [google_project_service.required]
}

# Cloud Run uses the authenticated Cloud SQL connector and no authorized
# network is configured. Public IPv4 remains a reviewed bridge until the
# production VPC/private-service-access increment is approved.
# Trivy 0.70 does not carry the literal ENCRYPTED_ONLY ssl_mode through both
# module callers, so the TLS control is documented and suppressed explicitly.
#trivy:ignore:AVD-GCP-0015:exp:2027-02-28
#trivy:ignore:AVD-GCP-0017:exp:2027-02-28
resource "google_sql_database_instance" "postgres" {
  project             = var.project_id
  name                = local.sql_instance_name
  region              = var.region
  database_version    = "POSTGRES_18"
  deletion_protection = var.deletion_protection

  settings {
    edition                     = "ENTERPRISE"
    tier                        = var.database_tier
    availability_type           = var.database_availability_type
    disk_type                   = "PD_SSD"
    disk_size                   = var.database_disk_size
    disk_autoresize             = true
    deletion_protection_enabled = var.deletion_protection
    connector_enforcement       = "REQUIRED"

    backup_configuration {
      enabled                        = true
      start_time                     = "03:00"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = var.backup_retention_count
        retention_unit   = "COUNT"
      }
    }

    ip_configuration {
      ipv4_enabled = true
      ssl_mode     = "ENCRYPTED_ONLY"
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
  secret_id = "${local.resource_prefix}-database-url"
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
  secret_id = "${local.resource_prefix}-fingerprint-salt"
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
  name         = "${local.resource_prefix}-firebase"
  display_name = "RentStage ${var.environment} Firebase Web API key"

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
  display_name    = "RentStage ${title(var.environment)}"
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

    phone_number {
      enabled            = false
      test_phone_numbers = {}
    }
  }

  authorized_domains = distinct(concat([
    "localhost",
    "${var.project_id}.firebaseapp.com",
    "${var.project_id}.web.app",
  ], var.additional_authorized_domains))

  # Cloud Run hostnames are generated at first deploy and then registered by
  # the environment deployment workflow through the Admin API.
  lifecycle {
    ignore_changes = [authorized_domains]
  }

  depends_on = [google_firebase_project.rentstage, google_project_service.required]
}

resource "google_secret_manager_secret" "firebase_api_key" {
  project   = var.project_id
  secret_id = "${local.resource_prefix}-firebase-api-key"
  replication {
    auto {}
  }
  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "firebase_api_key" {
  secret      = google_secret_manager_secret.firebase_api_key.id
  secret_data = data.google_firebase_web_app_config.rentstage.api_key
}

# Only secret containers are managed here. Meta credential values must be
# inserted out of band so they never appear in Terraform plans or state.
resource "google_secret_manager_secret" "meta" {
  for_each = var.enable_meta_secret_containers ? local.meta_secret_names : toset([])

  project   = var.project_id
  secret_id = "${local.resource_prefix}-${each.value}"
  replication {
    auto {}
  }
  depends_on = [google_project_service.required]
}

locals {
  api_project_roles = toset(concat(
    [
      "roles/cloudsql.client",
      "roles/logging.logWriter",
      "roles/serviceusage.serviceUsageConsumer",
    ],
    var.use_minimal_firebase_role ? [] : ["roles/firebaseauth.admin"]
  ))
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

resource "google_project_iam_custom_role" "api_runtime_firebase_session" {
  count = var.use_minimal_firebase_role ? 1 : 0

  project     = var.project_id
  role_id     = "rentstageApiFirebaseSession"
  title       = "RentStage API Firebase session runtime"
  description = "Read Firebase users and mint revocable session cookies without user or project administration."
  permissions = [
    "firebaseauth.users.createSession",
    "firebaseauth.users.get",
  ]
  stage = "GA"

  depends_on = [google_project_service.required]
}

resource "google_project_iam_member" "api_runtime" {
  for_each = local.api_project_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_project_iam_member" "api_runtime_firebase_session" {
  count = var.use_minimal_firebase_role ? 1 : 0

  project = var.project_id
  role    = google_project_iam_custom_role.api_runtime_firebase_session[0].name
  member  = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_project_iam_member" "web_runtime" {
  for_each = local.web_project_roles
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.web_runtime.email}"
}

# The staging deployment identity is WIF-bound to this repository, main, and
# its protected environment. Cloud Run lifecycle and Identity Platform domain
# administration currently have no narrower predefined roles for this flow.
#trivy:ignore:AVD-GCP-0007:exp:2027-02-28
resource "google_project_iam_member" "deploy" {
  for_each = var.enable_deploy_iam_bindings ? local.deploy_project_roles : toset([])
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${var.deploy_service_account_email}"
}

resource "google_service_account_iam_member" "deploy_acts_as_api" {
  count = var.enable_deploy_iam_bindings ? 1 : 0

  service_account_id = google_service_account.api_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${var.deploy_service_account_email}"
}

resource "google_service_account_iam_member" "deploy_acts_as_web" {
  count = var.enable_deploy_iam_bindings ? 1 : 0

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
  count = var.enable_deploy_iam_bindings ? 1 : 0

  project   = var.project_id
  secret_id = google_secret_manager_secret.firebase_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.deploy_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "api_meta" {
  for_each = google_secret_manager_secret.meta

  project   = var.project_id
  secret_id = each.value.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api_runtime.email}"
}
