# These declarations migrate the existing staging state into the shared module.
# They change Terraform addresses only and must not recreate remote resources.

moved {
  from = google_project_service.required
  to   = module.platform.google_project_service.required
}

moved {
  from = google_artifact_registry_repository.rentstage
  to   = module.platform.google_artifact_registry_repository.rentstage
}

moved {
  from = google_service_account.api_runtime
  to   = module.platform.google_service_account.api_runtime
}

moved {
  from = google_service_account.web_runtime
  to   = module.platform.google_service_account.web_runtime
}

moved {
  from = google_sql_database_instance.postgres
  to   = module.platform.google_sql_database_instance.postgres
}

moved {
  from = google_sql_database.rentstage
  to   = module.platform.google_sql_database.rentstage
}

moved {
  from = google_sql_user.rentstage
  to   = module.platform.google_sql_user.rentstage
}

moved {
  from = random_id.fingerprint_salt
  to   = module.platform.random_id.fingerprint_salt
}

moved {
  from = google_secret_manager_secret.database_url
  to   = module.platform.google_secret_manager_secret.database_url
}

moved {
  from = google_secret_manager_secret_version.database_url
  to   = module.platform.google_secret_manager_secret_version.database_url
}

moved {
  from = google_secret_manager_secret.fingerprint_salt
  to   = module.platform.google_secret_manager_secret.fingerprint_salt
}

moved {
  from = google_secret_manager_secret_version.fingerprint_salt
  to   = module.platform.google_secret_manager_secret_version.fingerprint_salt
}

moved {
  from = google_firebase_project.rentstage
  to   = module.platform.google_firebase_project.rentstage
}

moved {
  from = google_apikeys_key.firebase
  to   = module.platform.google_apikeys_key.firebase
}

moved {
  from = google_firebase_web_app.rentstage
  to   = module.platform.google_firebase_web_app.rentstage
}

moved {
  from = google_identity_platform_config.rentstage
  to   = module.platform.google_identity_platform_config.rentstage
}

moved {
  from = google_secret_manager_secret.firebase_api_key
  to   = module.platform.google_secret_manager_secret.firebase_api_key
}

moved {
  from = google_secret_manager_secret_version.firebase_api_key
  to   = module.platform.google_secret_manager_secret_version.firebase_api_key
}

moved {
  from = google_project_iam_member.api_runtime
  to   = module.platform.google_project_iam_member.api_runtime
}

moved {
  from = google_project_iam_member.web_runtime
  to   = module.platform.google_project_iam_member.web_runtime
}

moved {
  from = google_project_iam_member.deploy
  to   = module.platform.google_project_iam_member.deploy
}

moved {
  from = google_service_account_iam_member.deploy_acts_as_api
  to   = module.platform.google_service_account_iam_member.deploy_acts_as_api[0]
}

moved {
  from = google_service_account_iam_member.deploy_acts_as_web
  to   = module.platform.google_service_account_iam_member.deploy_acts_as_web[0]
}

moved {
  from = google_secret_manager_secret_iam_member.api_database_url
  to   = module.platform.google_secret_manager_secret_iam_member.api_database_url
}

moved {
  from = google_secret_manager_secret_iam_member.api_fingerprint_salt
  to   = module.platform.google_secret_manager_secret_iam_member.api_fingerprint_salt
}

moved {
  from = google_secret_manager_secret_iam_member.deploy_firebase_api_key
  to   = module.platform.google_secret_manager_secret_iam_member.deploy_firebase_api_key[0]
}
