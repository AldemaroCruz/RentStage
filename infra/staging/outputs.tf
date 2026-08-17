output "artifact_registry_repository" {
  value = google_artifact_registry_repository.rentstage.id
}

output "cloud_sql_connection_name" {
  value = google_sql_database_instance.postgres.connection_name
}

output "api_runtime_service_account" {
  value = google_service_account.api_runtime.email
}

output "web_runtime_service_account" {
  value = google_service_account.web_runtime.email
}

output "firebase_project_id" {
  value = var.project_id
}

output "firebase_web_app_id" {
  value = google_firebase_web_app.rentstage.app_id
}

output "database_url_secret" {
  value = google_secret_manager_secret.database_url.secret_id
}
