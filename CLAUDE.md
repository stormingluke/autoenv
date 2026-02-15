# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
task build        # Build binary (CGO_ENABLED=1 required)
task test         # Run tests with race detection
task lint         # Run golangci-lint
task              # Run lint + test
task install      # Build and install to GOPATH
task ci           # Run full Dagger CI pipeline (lint + test + build + goreleaser verify)
task release      # Dagger release snapshot → tag patch version → push
```

Run a single test: `CGO_ENABLED=1 go test -race -run TestFunctionName ./internal/app/...`

Dagger pipeline directly: `dagger call all --source=.`

## Rules

- All GitHub Actions workflow steps MUST use the Dagger pipeline. No direct `go build`, `golangci-lint`, or `goreleaser` invocations outside of Dagger containers.
- If a Dagger function runs successfully locally, it must also succeed in CI. The Dagger pipeline is the single source of truth for builds.
- `CGO_ENABLED=1` is required everywhere — go-libsql uses C bindings.
- Version info is injected via ldflags into `main.go` vars (`version`, `commit`, `date`).

## Architecture

Hexagonal architecture (ports & adapters). Inner layers never import outer layers.

```
cmd/           → CLI commands (cobra), DI wiring via bootstrap.go
internal/app/  → Service layer, orchestrates business logic via port interfaces
internal/port/ → Interfaces (ProjectRepository, SessionRepository, EnvLoader, ShellRenderer, SecretSyncer, ConfigStore)
internal/adapter/ → Implementations (sqlite/, shell/, envfile/, github/, config/)
internal/domain/  → Pure entities (Project, Session, EnvFile) + business logic (Diff, HashValue)
```

**Key patterns:**
- `cmd/bootstrap.go` has two bootstrap paths: `bootstrap()` (full — Turso + sessions DB) and `bootstrapLight()` (sessions DB only, used by `export` and `clear` on the hot path)
- `export` command is the hot path — called on every `cd` via shell hook. It uses `bootstrapLight()` to avoid Turso connection overhead, with a fast-exit when no `.env` and no active session
- Services depend on port interfaces, adapters implement them. `app.Deps` struct is the DI container
- Compile-time interface checks: `var _ port.Interface = (*Impl)(nil)`
- Two separate SQLite databases: `projects.db` (Turso-synced: projects + defaults) and `sessions.db` (local-only: sessions + session_keys)
- libsql quirk: separate `Exec()` calls per `CREATE TABLE` statement — combined statements fail
- SHA-256 truncated hashes for change detection without storing secrets in the database

## Dagger Pipeline (.dagger/main.go)

Go SDK module. Functions: `Lint`, `Test`, `Build`, `Release`, `All`. Uses `golang:1.25.7-bookworm` base image. Release uses `goreleaser-cross` image with osxcross for darwin/arm64 cross-compilation.
