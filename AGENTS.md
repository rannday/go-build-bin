# AGENTS.md

Repo purpose:

- `go-build-bin` is a Go-only release builder for other Go projects.
- Keep defaults aimed at common distribution targets:
  - `windows/amd64:zip`
  - `linux/amd64:tar.gz`
  - `linux/arm64:tar.gz`
  - `darwin/amd64:tar.gz`
  - `darwin/arm64:tar.gz`

Rules:

- Keep module path and import paths on `github.com/rannday/go-build-bin`.
- Keep repo root as tool entrypoint.
- Keep README examples generic. Do not hardcode old repo names.
- Keep `AGENTS.md` as durable agent guidance. Do not move these rules into README.
- Prefer narrow changes. Fix namespace, docs, defaults, and tests together when they drift.

Validation:

- Run `go test ./...` after code changes.
- Check output paths and archive names in tests when target defaults change.
