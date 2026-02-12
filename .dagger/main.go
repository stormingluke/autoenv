// Autoenv CI pipeline: lint, test, build, and release
package main

import (
	"context"
	"fmt"

	"dagger/autoenv/internal/dagger"
)

const zigVersion = "0.14.0"

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

// Build compiles the binary for a given platform
func (m *Autoenv) Build(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="linux"
	goos string,
	// +optional
	// +default="amd64"
	goarch string,
) *dagger.File {
	ctr := m.base(source).
		WithEnvVariable("GOOS", goos).
		WithEnvVariable("GOARCH", goarch)

	// Cross-compile darwin from linux using zig
	if goos == "darwin" && goarch == "arm64" {
		ctr = m.withZig(ctr).
			WithEnvVariable("GOOS", goos).
			WithEnvVariable("GOARCH", goarch).
			WithEnvVariable("CC", "zig cc -target aarch64-macos").
			WithEnvVariable("CXX", "zig c++ -target aarch64-macos")
	}

	outputName := fmt.Sprintf("autoenv-%s-%s", goos, goarch)
	return ctr.
		WithExec([]string{"go", "build", "-trimpath", "-o", outputName, "."}).
		File(fmt.Sprintf("/src/%s", outputName))
}

// BuildAll compiles binaries for all supported platforms (linux/amd64 + darwin/arm64)
func (m *Autoenv) BuildAll(ctx context.Context, source *dagger.Directory) *dagger.Directory {
	return dag.Directory().
		WithFile("autoenv-linux-amd64", m.Build(ctx, source, "linux", "amd64")).
		WithFile("autoenv-darwin-arm64", m.Build(ctx, source, "darwin", "arm64"))
}

// Release runs GoReleaser to create a GitHub release
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

	ctr := m.withZig(
		dag.Container().
			From("golang:1.25-bookworm").
			WithEnvVariable("CGO_ENABLED", "1"),
	)

	return ctr.
		WithExec([]string{"go", "install", "github.com/goreleaser/goreleaser/v2@latest"}).
		WithDirectory("/workspace", source).
		WithWorkdir("/workspace").
		WithSecretVariable("GITHUB_TOKEN", githubToken).
		WithExec(args).
		Stdout(ctx)
}

// All runs lint, test, and build for all platforms
func (m *Autoenv) All(ctx context.Context, source *dagger.Directory) (string, error) {
	if _, err := m.Lint(ctx, source); err != nil {
		return "", err
	}
	if _, err := m.Test(ctx, source); err != nil {
		return "", err
	}
	_ = m.BuildAll(ctx, source)
	return "All CI checks passed.", nil
}

// base returns a Go build container without zig (used for lint/test/native builds)
func (m *Autoenv) base(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:1.25-bookworm").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithEnvVariable("CGO_ENABLED", "1")
}

// withZig installs zig into a container for cross-compilation
func (m *Autoenv) withZig(ctr *dagger.Container) *dagger.Container {
	zigURL := fmt.Sprintf("https://ziglang.org/download/%s/zig-linux-x86_64-%s.tar.xz", zigVersion, zigVersion)
	return ctr.
		WithExec([]string{"sh", "-c",
			fmt.Sprintf("curl -fsSL %s | tar xJ -C /opt && ln -sf /opt/zig-linux-x86_64-%s/zig /usr/local/bin/zig", zigURL, zigVersion)})
}
