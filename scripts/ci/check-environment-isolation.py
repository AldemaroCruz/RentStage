#!/usr/bin/env python3
"""Guard the staging/production Terraform and workflow isolation contract."""

from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def read(relative_path: str) -> str:
    return (ROOT / relative_path).read_text(encoding="utf-8")


staging_workflow = read(".github/workflows/infra-staging.yml")
production_plan_workflow = read(".github/workflows/infra-production-plan.yml")
production_apply_workflow = read(".github/workflows/infra-production-apply.yml")
staging_root = read("infra/staging/main.tf")
production_root = read("infra/production/main.tf")
platform_module = read("infra/modules/rentstage-platform/main.tf")
staging_moves = read("infra/staging/moved.tf")
dependabot = read(".github/dependabot.yml")
production_bootstrap = read("scripts/bootstrap-gcp-production.sh")
production_apply_access = read("scripts/gcp/production-apply-access.sh")
production_plan_guard = read("scripts/ci/check_production_apply_plan.py")

apply_admin_roles = (
    "roles/artifactregistry.admin",
    "roles/cloudsql.admin",
    "roles/firebase.admin",
    "roles/iam.roleAdmin",
    "roles/iam.serviceAccountAdmin",
    "roles/identityplatform.admin",
    "roles/resourcemanager.projectIamAdmin",
    "roles/secretmanager.admin",
    "roles/serviceusage.apiKeysAdmin",
    "roles/serviceusage.serviceUsageAdmin",
)

assertions = {
    "staging uses its own state prefix": 'prefix=rentstage/staging' in staging_workflow,
    "production plan uses its own state prefix": 'prefix=rentstage/production' in production_plan_workflow,
    "production apply uses the same state prefix": 'prefix=rentstage/production' in production_apply_workflow,
    "production plan runs in its protected environment": "environment: production" in production_plan_workflow,
    "production apply runs in its protected environment": "environment: production" in production_apply_workflow,
    "production workflows read their own database secret": all(
        "secrets.PRODUCTION_DATABASE_PASSWORD" in workflow
        for workflow in (production_plan_workflow, production_apply_workflow)
    ),
    "production workflows never read the staging database secret": all(
        "STAGING_DATABASE_PASSWORD" not in workflow
        for workflow in (production_plan_workflow, production_apply_workflow)
    ),
    "production plan workflow contains no Terraform apply": "terraform apply" not in production_plan_workflow,
    "production apply is manual only": "workflow_dispatch:" in production_apply_workflow
    and "push:" not in production_apply_workflow
    and "schedule:" not in production_apply_workflow,
    "production apply uses its dedicated identity": all(
        variable in production_apply_workflow
        for variable in (
            "GCP_INFRA_APPLY_WIF_PROVIDER",
            "GCP_INFRA_APPLY_SERVICE_ACCOUNT",
        )
    ),
    "production apply requires gate and two confirmations": all(
        marker in production_apply_workflow
        for marker in (
            "PRODUCTION_INFRA_APPLY_ENABLED",
            "confirm_project_id",
            "APPLY-PRODUCTION",
        )
    ),
    "production apply uses the exact saved plan": "terraform apply -input=false -lock-timeout=5m -auto-approve tfplan" in production_apply_workflow,
    "production apply invokes the JSON plan guard": "check_production_apply_plan.py" in production_apply_workflow,
    "production apply never uploads its sensitive plan": "upload-artifact" not in production_apply_workflow,
    "production root identifies itself": 'environment                   = "production"' in production_root,
    "staging root identifies itself": 'environment                   = "staging"' in staging_root,
    "production prepares Meta secret containers": "enable_meta_secret_containers = true" in production_root,
    "staging does not prepare Meta secret containers": "enable_meta_secret_containers = false" in staging_root,
    "production has no deployment IAM bindings": "enable_deploy_iam_bindings    = false" in production_root,
    "staging preserves its deployment IAM bindings": "enable_deploy_iam_bindings    = true" in staging_root,
    "production enables the minimal Firebase runtime role": "use_minimal_firebase_role     = true" in production_root,
    "Firebase runtime custom role has only session permissions": all(
        permission in platform_module
        for permission in (
            '"firebaseauth.users.createSession"',
            '"firebaseauth.users.get"',
        )
    )
    and "firebaseauth.users.update" not in platform_module
    and "firebaseauth.users.delete" not in platform_module,
    "production plan identity is project read-only": all(
        role in production_bootstrap
        for role in ("roles/viewer", "roles/serviceusage.serviceUsageConsumer")
    ),
    "production bootstrap grants no infrastructure admin role": not any(
        role in production_bootstrap for role in apply_admin_roles
    ),
    "production apply permissions are complete and temporary": all(
        role in production_apply_access for role in apply_admin_roles
    )
    and all(operation in production_apply_access for operation in ("grant", "revoke", "status")),
    "production apply has an isolated WIF pool": "github-actions-production-apply" in production_bootstrap,
    "production apply WIF is restricted to its workflow": "infra-production-apply.yml@refs/heads/main" in production_bootstrap,
    "reserved production deploy identity has no WIF binding loop": 'for service_account in "$INFRA_SA" "$DEPLOY_SA"' not in production_bootstrap,
    "production plan guard rejects every mutating action except create": 'SAFE_ACTIONS = {"create", "no-op", "read"}' in production_plan_guard,
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
