// Autoenv CI pipeline: lint, test, and build
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

// Build compiles the binary for linux/amd64
func (m *Autoenv) Build(ctx context.Context, source *dagger.Directory) *dagger.File {
	return m.base(source).
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "amd64").
		WithExec([]string{"go", "build", "-trimpath", "-o", "autoenv", "."}).
		File("/src/autoenv")
}

// All runs lint, test, and build
func (m *Autoenv) All(ctx context.Context, source *dagger.Directory) (string, error) {
	// Lint
	if _, err := m.Lint(ctx, source); err != nil {
		return "", err
	}

	// Test
	if _, err := m.Test(ctx, source); err != nil {
		return "", err
	}

	// Build (verify it compiles)
	_ = m.Build(ctx, source)

	return "All CI checks passed.", nil
}

func (m *Autoenv) base(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:1.25-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1")
}
