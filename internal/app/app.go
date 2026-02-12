package app

import "github.com/stormingluke/autoenv/internal/port"

type App struct {
	Export    *ExportService
	Load     *LoadService
	Clear    *ClearService
	List     *ListService
	Sync     *SyncService
	Configure *ConfigureService
}

type Deps struct {
	Projects  port.ProjectRepository
	Sessions  port.SessionRepository
	EnvLoader port.EnvLoader
	Shell     port.ShellRenderer
	Syncer    port.SecretSyncer
	Config    port.ConfigStore
}

func New(d Deps) *App {
	return &App{
		Export:    &ExportService{projects: d.Projects, sessions: d.Sessions, envLoader: d.EnvLoader, shell: d.Shell},
		Load:     &LoadService{projects: d.Projects, sessions: d.Sessions, envLoader: d.EnvLoader, shell: d.Shell},
		Clear:    &ClearService{sessions: d.Sessions, shell: d.Shell},
		List:     &ListService{projects: d.Projects},
		Sync:     &SyncService{projects: d.Projects, envLoader: d.EnvLoader, syncer: d.Syncer, config: d.Config},
		Configure: &ConfigureService{config: d.Config},
	}
}
