# RentStage security policy

## Supported environments

- `local`: Docker Compose, PostgreSQL, and Firebase Authentication Emulator.
- `staging`: Cloud Run, Cloud SQL, Identity Platform/Firebase Auth, Secret Manager, and GitHub Actions.
- `production`: intentionally not deployed by v0.13.0.

## Reporting a vulnerability

Do not open a public issue containing credentials, customer data, tokens, DTE payloads, or exploit details. Report the issue privately to the repository owner and include:

- affected version and component;
- reproduction steps;
- expected and observed behavior;
- impact and any known workaround.

## Credential rules

- GitHub Actions authenticates to Google Cloud with Workload Identity Federation; service-account JSON keys are prohibited.
- Runtime secrets belong in Google Secret Manager.
- `.env`, Terraform state, database dumps, private keys, and generated Google credentials must never be committed.
- DTE remains `MOCK / TEST` in staging until a real issuer is authorized and homologated.

## Required checks

Pull requests should pass unit tests, race detection, type checking, production builds, migration validation, secret scanning, dependency vulnerability scans, static security analysis, container scanning, and the complete local smoke suite.
