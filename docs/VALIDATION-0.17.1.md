# RentStage v0.17.1 validation record

## Release consistency

- [ ] `VERSION`, `apps/web/package.json`, README, dashboard release label, and changelog report `0.17.1`.
- [ ] `git diff --check` passes.
- [ ] Workflow YAML parsing and repository contracts pass.
- [ ] Sensitive-file policy passes and no Terraform plan/state or GCP credential is tracked.

## Terraform

- [ ] `terraform fmt -check -recursive` passes for both roots and the shared module.
- [ ] Backend-free initialization and validation pass for staging and production.
- [ ] Staging remote plan is `0 add / 0 change / 0 destroy`.
- [ ] Production plan contains only expected creates and no staging reference.
- [ ] Production API runtime custom role contains exactly user lookup and session creation permissions.
- [ ] `enable_deploy_iam_bindings` remains false in production.

## Identity and workflow isolation

- [ ] Plan SA remains read-only outside its state bucket.
- [ ] Apply SA uses a separate WIF pool and its provider condition names the exact apply workflow on `main` in `production`.
- [ ] Apply SA receives no project mutation role from bootstrap.
- [ ] Reserved deploy SA has no project role and no GitHub impersonation binding.
- [ ] Apply workflow is manual, protected, serialized, and requires gate, exact project ID, and confirmation phrase.
- [ ] Apply and plan workflows use only the production database secret and production state prefix.
- [ ] No workflow uploads `tfplan` or its JSON form.

## Plan safety tests

- [ ] A create-only module plan passes.
- [ ] Update is rejected.
- [ ] Replacement and destroy are rejected.
- [ ] Wrong input project is rejected.
- [ ] Cross-project resource is rejected.
- [ ] Root-level resource outside `module.platform` is rejected.

## Manual first-apply controls

- [ ] Current GCP cost estimate and budget alerts were reviewed.
- [ ] Temporary roles were granted immediately before the workflow.
- [ ] Protected Environment reviewer approved the expected commit.
- [ ] The exact saved plan was applied in the same job.
- [ ] Repository apply gate was reset to false after the run.
- [ ] Temporary apply roles were revoked after the run.
- [ ] Follow-up production plan is `0 add / 0 change / 0 destroy`.
- [ ] Application production deploy and Meta credential versions remain absent.
