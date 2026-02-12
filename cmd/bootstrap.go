package cmd

import (
	"fmt"
	"io"

	"github.com/stormingluke/autoenv/internal/adapter/config"
	"github.com/stormingluke/autoenv/internal/adapter/envfile"
	"github.com/stormingluke/autoenv/internal/adapter/github"
	"github.com/stormingluke/autoenv/internal/adapter/shell"
	"github.com/stormingluke/autoenv/internal/adapter/sqlite"
	"github.com/stormingluke/autoenv/internal/app"
)

type closers []io.Closer

func (c *closers) Add(closer io.Closer) { *c = append(*c, closer) }
func (c closers) CloseAll() {
	for i := len(c) - 1; i >= 0; i-- {
		c[i].Close()
	}
}

type bootstrapResult struct {
	app   *app.App
	turso *sqlite.TursoDB
	cc    closers
}

func bootstrap() (*bootstrapResult, error) {
	var cc closers

	cfg := config.Load()
	if err := cfg.EnsureDir(); err != nil {
		return nil, fmt.Errorf("ensure config dir: %w", err)
	}

	turso, err := sqlite.OpenTurso(cfg.ProjectsDBPath, cfg.TursoURL, cfg.TursoAuthToken)
	if err != nil {
		return nil, fmt.Errorf("open projects db: %w", err)
	}
	cc.Add(turso)

	sessDB, sessCloser, err := sqlite.OpenSessionsDB(cfg.SessionsDBPath)
	if err != nil {
		cc.CloseAll()
		return nil, fmt.Errorf("open sessions db: %w", err)
	}
	cc.Add(sessCloser)

	projectRepo := sqlite.NewProjectRepo(turso.DB)
	sessionRepo := sqlite.NewSessionRepo(sessDB)
	defaultsRepo := sqlite.NewDefaultsRepo(turso.DB)

	a := app.New(app.Deps{
		Projects:  projectRepo,
		Sessions:  sessionRepo,
		EnvLoader: envfile.NewLoader(),
		Shell:     shell.NewRenderer(),
		Syncer:    github.NewSecretSyncer(),
		Config:    defaultsRepo,
	})

	return &bootstrapResult{app: a, turso: turso, cc: cc}, nil
}

func bootstrapLight() (*app.App, closers, error) {
	var cc closers

	cfg := config.Load()
	if err := cfg.EnsureDir(); err != nil {
		return nil, nil, fmt.Errorf("ensure config dir: %w", err)
	}

	sessDB, sessCloser, err := sqlite.OpenSessionsDB(cfg.SessionsDBPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open sessions db: %w", err)
	}
	cc.Add(sessCloser)

	a := app.New(app.Deps{
		Sessions:  sqlite.NewSessionRepo(sessDB),
		Shell:     shell.NewRenderer(),
		EnvLoader: envfile.NewLoader(),
	})

	return a, cc, nil
}
