# Upgrade to RentStage v0.18.1

1. Start from a clean committed v0.18.0 worktree.
2. Apply the supplied patch with the installer.
3. Keep `META_OUTBOUND_ENABLED=true` only in local Compose. Do not add it to staging or production.
4. For a public review deployment, configure `RENTSTAGE_SUPPORT_EMAIL` in the staging GitHub Environment. Local builds can set `NEXT_PUBLIC_SUPPORT_EMAIL` directly.
5. Run the repository, Go, frontend, migration, and local Meta contract validations.
6. Review migration `015_meta_application_readiness.sql`, commit, and push.

No Terraform apply, production deployment, real sender registration, or credential creation is required by this increment.
