output "artifact_registry_repository" {
  value = module.platform.artifact_registry_repository
}

output "cloud_sql_connection_name" {
  value = module.platform.cloud_sql_connection_name
}

output "cloud_sql_instance_name" {
  value = module.platform.cloud_sql_instance_name
}

output "api_runtime_service_account" {
  value = module.platform.api_runtime_service_account
}

output "web_runtime_service_account" {
  value = module.platform.web_runtime_service_account
}

output "firebase_project_id" {
  value = module.platform.firebase_project_id
}

output "firebase_web_app_id" {
  value = module.platform.firebase_web_app_id
}

output "database_url_secret" {
  value = module.platform.database_url_secret
}

output "firebase_api_key_secret" {
  value = module.platform.firebase_api_key_secret
}

output "meta_secret_ids" {
  description = "Empty production Meta secret containers. Populate values outside Terraform."
  value       = module.platform.meta_secret_ids
}
