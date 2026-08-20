# Deferred production apply record — v0.18.0

The first production infrastructure apply is intentionally archived for a later decision. This is a pause, not an approval and not a partial deployment.

## Last reviewed evidence

- Staging plan after the shared-module move: `0 add / 0 change / 0 destroy`.
- Production review-only plan: expected create-only resources, with no staging reference and no update or destroy.
- Production project: isolated from staging with its own state, identities, database inputs, and empty Meta secret containers.
- Protected apply workflow and just-in-time permission procedure exist, but are not invoked by v0.18.0.

## Controls that must remain closed

- `PRODUCTION_INFRA_APPLY_ENABLED=false`
- `PRODUCTION_DEPLOY_ENABLED=false`
- temporary apply roles revoked or absent
- reserved deploy identity without project roles or GitHub impersonation
- no Secret Manager versions for Meta credentials
- no production Cloud Run application deployment

## Resume criteria

Before resuming, rerun both remote plans from the exact intended commit, verify staging is still zero-change, review current GCP monthly cost, approve the create-only production inventory, confirm the protected GitHub Environment reviewer, grant temporary apply roles immediately before the run, and revoke them immediately afterward.

Source development, local migrations, and the local Meta harness do not require or imply a production apply.
