// Autoenv CI pipeline: lint, test, build, and release
package main

import (
	"context"

	"dagger/autoenv/internal/dagger"
)

type Autoenv struct{}

// Lint runs golangci-lint on the source code
func (m *Autoenv) Lint(ctx context.Context, source *dagger.Directory) (string, error) {
	return m.base(source).
		WithExec([]string{"go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"}).
		WithExec([]string{"golangci-lint", "run", "./..."}).
		Stdout(ctx)
}

// Test runs go test with race detection
func (m *Autoenv) Test(ctx context.Context, source *dagger.Directory) (string, error) {
	return m.base(source).
		WithExec([]string{"go", "test", "-race", "-count=1", "./..."}).
		Stdout(ctx)
}

// Build compiles the linux/amd64 binary
func (m *Autoenv) Build(ctx context.Context, source *dagger.Directory) *dagger.File {
	return m.amd64(source).
		WithExec([]string{"go", "build", "-trimpath", "-o", "autoenv", "."}).
		File("/src/autoenv")
}

// Release runs GoReleaser to create a GitHub release (linux/amd64)
func (m *Autoenv) Release(
	ctx context.Context,
	source *dagger.Directory,
	githubToken *dagger.Secret,
	// +optional
	// +default=false
	snapshot bool,
) (string, error) {
	args := []string{"goreleaser", "release", "--clean"}
	if snapshot {
		args = append(args, "--snapshot")
	}

	return m.amd64(source).
		WithExec([]string{"go", "install", "github.com/goreleaser/goreleaser/v2@latest"}).
		WithSecretVariable("GITHUB_TOKEN", githubToken).
		WithExec(args).
		Stdout(ctx)
}

// All runs lint, test, build, and verifies goreleaser
func (m *Autoenv) All(ctx context.Context, source *dagger.Directory) (string, error) {
	if _, err := m.Lint(ctx, source); err != nil {
		return "", err
	}
	if _, err := m.Test(ctx, source); err != nil {
		return "", err
	}
	if _, err := m.Build(ctx, source).Sync(ctx); err != nil {
		return "", err
	}
	// Verify goreleaser config and build without publishing
	ctr := m.amd64(source).
		WithExec([]string{"go", "install", "github.com/goreleaser/goreleaser/v2@latest"})
	if _, err := ctr.WithExec([]string{"goreleaser", "check"}).Stdout(ctx); err != nil {
		return "", err
	}
	if _, err := ctr.WithExec([]string{"goreleaser", "build", "--snapshot", "--clean"}).Stdout(ctx); err != nil {
		return "", err
	}
	return "All CI checks passed.", nil
}

// base returns a Go container on the native platform (fast for lint/test)
func (m *Autoenv) base(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:1.25-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1")
}

// amd64 returns a Go container pinned to linux/amd64 for native compilation
func (m *Autoenv) amd64(source *dagger.Directory) *dagger.Container {
	return dag.Container(dagger.ContainerOpts{Platform: "linux/amd64"}).
		From("golang:1.25-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1")
}
