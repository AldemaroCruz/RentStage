#!/usr/bin/env python3
"""Guard the staging/production Terraform and workflow isolation contract."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def read(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


staging_workflow = read(".github/workflows/infra-staging.yml")
production_workflow = read(".github/workflows/infra-production-plan.yml")
staging_root = read("infra/staging/main.tf")
production_root = read("infra/production/main.tf")
staging_moves = read("infra/staging/moved.tf")
dependabot = read(".github/dependabot.yml")
production_bootstrap = read("scripts/bootstrap-gcp-production.sh")

assertions = {
    "staging uses its own state prefix": 'prefix=rentstage/staging' in staging_workflow,
    "production uses its own state prefix": 'prefix=rentstage/production' in production_workflow,
    "production runs in its protected environment": "environment: production" in production_workflow,
    "production reads its own database secret": "secrets.PRODUCTION_DATABASE_PASSWORD" in production_workflow,
    "production never reads the staging database secret": "STAGING_DATABASE_PASSWORD" not in production_workflow,
    "production workflow contains no Terraform apply": "terraform apply" not in production_workflow,
    "production workflow exposes no apply operation": "operation:" not in production_workflow,
    "production root identifies itself": 'environment                   = "production"' in production_root,
    "staging root identifies itself": 'environment                   = "staging"' in staging_root,
    "production prepares Meta secret containers": "enable_meta_secret_containers = true" in production_root,
    "staging does not prepare Meta secret containers": "enable_meta_secret_containers = false" in staging_root,
    "production has no deployment IAM bindings": "enable_deploy_iam_bindings    = false" in production_root,
    "staging preserves its deployment IAM bindings": "enable_deploy_iam_bindings    = true" in staging_root,
    "production plan identity is project read-only": all(
        role in production_bootstrap
        for role in ("roles/viewer", "roles/serviceusage.serviceUsageConsumer")
    ),
    "production bootstrap grants no infrastructure admin role": not any(
        role in production_bootstrap
        for role in (
            "roles/artifactregistry.admin",
            "roles/cloudsql.admin",
            "roles/firebase.admin",
            "roles/iam.serviceAccountAdmin",
            "roles/identityplatform.admin",
            "roles/resourcemanager.projectIamAdmin",
            "roles/secretmanager.admin",
            "roles/serviceusage.apiKeysAdmin",
            "roles/serviceusage.serviceUsageAdmin",
        )
    ),
    "reserved production deploy identity has no WIF binding loop": 'for service_account in "$INFRA_SA" "$DEPLOY_SA"' not in production_bootstrap,
    "staging state migration declarations exist": staging_moves.count("moved {") >= 20,
    "Dependabot monitors both Terraform roots": all(
        directory in dependabot
        for directory in ("/infra/staging", "/infra/production")
    ),
}

failures = [description for description, passed in assertions.items() if not passed]
if failures:
    for failure in failures:
        print(f"Environment isolation failed: {failure}")
    raise SystemExit(1)

print(f"Environment isolation passed ({len(assertions)} contracts).")
