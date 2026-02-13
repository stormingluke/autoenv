// Autoenv CI pipeline: lint, test, build, and release
package main

import (
	"context"
	"fmt"
	"time"

	"dagger/autoenv/internal/dagger"
)

const goVersion = "1.25.7"

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

// Build compiles the linux/amd64 binary with optional version info
func (m *Autoenv) Build(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="dev"
	version string,
	// +optional
	// +default="none"
	commit string,
) *dagger.File {
	ldflags := fmt.Sprintf("-s -w -X main.version=%s -X main.commit=%s -X main.date=%s",
		version, commit, time.Now().UTC().Format(time.RFC3339))
	return m.amd64(source).
		WithExec([]string{"go", "build", "-trimpath", "-ldflags", ldflags, "-o", "autoenv", "."}).
		File("/src/autoenv")
}

// Release runs GoReleaser to create a GitHub release (linux/amd64 + darwin/arm64)
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

	return m.releaseCtr(source).
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
	if _, err := m.Build(ctx, source, "dev", "none").Sync(ctx); err != nil {
		return "", err
	}
	// Verify goreleaser config and linux build without publishing
	ctr := m.amd64(source).
		WithExec([]string{"go", "install", "github.com/goreleaser/goreleaser/v2@latest"})
	if _, err := ctr.WithExec([]string{"goreleaser", "check"}).Stdout(ctx); err != nil {
		return "", err
	}
	if _, err := ctr.WithExec([]string{"goreleaser", "build", "--id", "autoenv-linux", "--snapshot", "--clean"}).Stdout(ctx); err != nil {
		return "", err
	}
	return "All CI checks passed.", nil
}

// base returns a Go container on the native platform (fast for lint/test)
func (m *Autoenv) base(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:"+goVersion+"-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1")
}

// amd64 returns a Go container pinned to linux/amd64 for native compilation
func (m *Autoenv) amd64(source *dagger.Directory) *dagger.Container {
	return dag.Container(dagger.ContainerOpts{Platform: "linux/amd64"}).
		From("golang:"+goVersion+"-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1")
}

// releaseCtr returns a goreleaser-cross container with osxcross for cross-platform releases
func (m *Autoenv) releaseCtr(source *dagger.Directory) *dagger.Container {
	return dag.Container(dagger.ContainerOpts{Platform: "linux/amd64"}).
		From("ghcr.io/goreleaser/goreleaser-cross:v1.25.7-v2.13.3").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1")
}
