# autoenv

## Rules

- All GitHub Actions workflow steps MUST use the Dagger pipeline. No direct `go build`, `golangci-lint`, or `goreleaser` invocations outside of Dagger containers.
- If a Dagger function runs successfully locally, it must also succeed in CI. The Dagger pipeline is the single source of truth for builds.
