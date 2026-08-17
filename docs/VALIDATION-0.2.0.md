# Validation notes for v0.2.0

The following checks were completed before packaging:

- `gofmt` across the API source.
- Go package compilation and `go test ./...` using a local compile-time pgx contract stub.
- `go vet ./...` using the same pgx contract stub.
- Customer normalization unit tests.
- Quote totals and duplicate-resource validation unit tests.
- Strict TypeScript static checking with local React/Next declaration stubs.
- CSS block-balance validation.
- JSON parsing for `package.json` and `tsconfig.json`.
- YAML parsing for `compose.yaml`.
- Resolution check for local `@/` imports and expected routes.
- Review for accidental generated files and obvious secret material.

The package was not executed with `docker compose up` in the packaging environment because no Docker daemon or external package registry resolution was available there. The intended final integration test is therefore the local Docker rebuild documented in `UPGRADE-0.2.0.md`.
