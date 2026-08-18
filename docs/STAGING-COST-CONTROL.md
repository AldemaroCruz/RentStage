# RentStage staging cost control

RentStage keeps Cloud Run at zero minimum instances, so an idle web or API service can scale to zero while retaining its hostname and IAM configuration. The main fixed staging runtime cost is Cloud SQL. The **Staging Cost Control** workflow changes only its activation policy:

- `status` reads the current state without changing it;
- `pause` sets Cloud SQL activation policy to `NEVER`;
- `resume` restores activation policy to `ALWAYS`.

No database, secret, container image, Terraform state, service account, or Cloud Run service is deleted.

## Safety model

- The workflow runs manually from `main` only.
- It uses the existing `staging` GitHub Environment and keyless infrastructure service account.
- `pause` and `resume` require `confirm=true`.
- `pause` is rejected unless repository variable `STAGING_DEPLOY_ENABLED` is exactly `false`.
- Cost control, Terraform changes, and application deployment share concurrency group `rentstage-staging-change`.
- The application pipeline fails before image builds if its deployment gate is enabled while Cloud SQL remains paused.

If the `staging` Environment has required reviewers, the same approval applies to these operations.

## Pause staging

First disable automatic application deployment:

```bash
export REPO="AldemaroCruz/RentStage"

gh variable set STAGING_DEPLOY_ENABLED \
  --repo "$REPO" \
  --body "false"
```

Then use **Actions → Staging Cost Control → Run workflow**, select `pause`, enable `confirm`, and run it from `main`.

Equivalent GitHub CLI command:

```bash
gh workflow run staging-cost-control.yml \
  --repo "$REPO" \
  --ref main \
  -f operation=pause \
  -f confirm=true
```

While paused, login and all database-backed demo operations are unavailable. Cloud Run remains deployed but has no minimum-instance reservation.

## Inspect status

Run `status` at any time:

```bash
gh workflow run staging-cost-control.yml \
  --repo "$REPO" \
  --ref main \
  -f operation=status \
  -f confirm=false
```

The job log and summary report the Cloud SQL state and activation policy, the automatic deployment gate, and the deployed Cloud Run services.

## Resume staging

Start the database first:

```bash
gh workflow run staging-cost-control.yml \
  --repo "$REPO" \
  --ref main \
  -f operation=resume \
  -f confirm=true
```

Wait for the workflow to complete successfully. Then re-enable automatic application deployment:

```bash
gh variable set STAGING_DEPLOY_ENABLED \
  --repo "$REPO" \
  --body "true"
```

Run **RentStage CI/CD** manually if a fresh deployment and full smoke validation are needed immediately. Otherwise the next successful push to `main` will deploy normally.

## Expected billing behavior

Stopping Cloud SQL suspends instance charges and leaves the data intact. Storage and IP-related charges continue; backups and other retained Google Cloud resources may also continue to incur charges. Cloud Run can scale to zero, but requests to its public URL can still start an instance and database-backed requests will fail while Cloud SQL is paused.

The existing monthly budget is an alert, not a hard spending cap. Keep reviewing Google Cloud Billing even while staging is paused.

## Terraform and recovery

The activation policy is an operational override. A later Terraform plan can report it as drift and a Terraform apply can restore the configured/default active policy. Prefer resuming Cloud SQL before applying infrastructure changes.

If the workflow fails, inspect the current policy directly:

```bash
gcloud sql instances describe rentstage-staging-postgres \
  --project banded-nimbus-505914-s3 \
  --format='table(name,state,settings.activationPolicy)'
```

Emergency manual recovery:

```bash
gcloud sql instances patch rentstage-staging-postgres \
  --project banded-nimbus-505914-s3 \
  --activation-policy=ALWAYS \
  --quiet
```

Do not delete the Cloud SQL instance to save costs. Deletion is a different, destructive operation and is not part of this workflow.
