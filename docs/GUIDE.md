# Autoenv Developer Guide

## 1. Introduction

This guide is for contributors and developers who want to understand or extend autoenv. It provides a deep dive into the architecture, codebase structure, and implementation details.

For usage and installation instructions, see the [README.md](../README.md).

## 2. Architecture Overview

Autoenv follows **hexagonal architecture** (also known as ports and adapters pattern). This separates business logic from infrastructure concerns and makes the codebase testable and adaptable.

### Architecture Layers

```
cmd/ (CLI)
    ↓
app/ (services)
    ↓
port/ (interfaces) ← adapter/ (implementations)
    ↑
domain/ (entities)
```

### Dependency Rule

**Inner layers never import outer layers.**

- `domain/` has ZERO infrastructure imports (only stdlib: `crypto/sha256`, `fmt`, `errors`)
- `port/` defines interfaces, depends only on `domain/`
- `adapter/` implements interfaces, depends on `port/` and `domain/`
- `app/` orchestrates business logic, depends on `port/` and `domain/`
- `cmd/` wires everything together, depends on all layers

This ensures:
- Domain logic is pure and testable
- Infrastructure can be swapped without changing business logic
- Clear separation of concerns

## 3. Directory Structure

```
autoenv/
├── main.go                           # Entry point, sets version and calls cmd.Execute()
├── go.mod                            # Go module dependencies
├── go.sum                            # Dependency checksums
├── Taskfile.yml                      # Local dev tasks (build, test, lint)
├── .goreleaser.yaml                  # Release configuration
├── .github/
│   └── workflows/
│       └── ci.yml                    # GitHub Actions CI/CD
├── .dagger/
│   └── main.go                       # Dagger CI pipeline (lint, test, build, release)
├── cmd/                              # Command layer (CLI commands)
│   ├── root.go                       # Root cobra command
│   ├── bootstrap.go                  # DI bootstrapping (full + light)
│   ├── export.go                     # export command (called by shell hook)
│   ├── load.go                       # load command (register project)
│   ├── clear.go                      # clear command (unset all vars)
│   ├── list.go                       # list command (show projects)
│   ├── sync.go                       # sync command (secrets + Turso)
│   ├── configure.go                  # configure command (manage defaults)
│   └── hook.go                       # hook command (output shell hook)
└── internal/
    ├── domain/                       # Domain layer (pure business entities)
    │   ├── project.go                # Project entity
    │   ├── session.go                # Session, SessionKey entities + KeyNames() helper
    │   ├── envfile.go                # EnvFile, DiffResult, Diff(), HashValue(), KeyHashes()
    │   └── errors.go                 # Sentinel errors + DefaultSetting type
    ├── port/                         # Port layer (interfaces)
    │   ├── repository.go             # ProjectReader, ProjectWriter, ProjectRepository, SessionRepository
    │   ├── envloader.go              # EnvLoader interface
    │   ├── shellrenderer.go          # ShellRenderer interface
    │   ├── configstore.go            # ConfigStore interface
    │   └── secretsync.go             # SecretSyncer interface
    ├── adapter/                      # Adapter layer (implementations)
    │   ├── sqlite/                   # SQLite/Turso adapter
    │   │   ├── turso.go              # TursoDB, OpenTurso(), Sync(), Close()
    │   │   ├── local.go              # OpenLocal(), OpenSessionsDB() with WAL mode
    │   │   ├── migrations.go         # Schema creation (projects + sessions)
    │   │   ├── project_repo.go       # ProjectRepo (implements ProjectRepository)
    │   │   ├── session_repo.go       # SessionRepo (implements SessionRepository)
    │   │   └── defaults_repo.go      # DefaultsRepo (implements ConfigStore)
    │   ├── shell/                    # Shell adapter
    │   │   ├── hook.go               # HookScript() for zsh/bash
    │   │   └── renderer.go           # Renderer (implements ShellRenderer)
    │   ├── envfile/                  # .env file adapter
    │   │   └── loader.go             # Loader (implements EnvLoader)
    │   ├── config/                   # Config adapter
    │   │   └── config.go             # XDG-compliant config directory resolution
    │   └── github/                   # GitHub adapter
    │       └── secrets.go            # SecretSyncer (implements SecretSyncer via gh CLI)
    └── app/                          # Application layer (services)
        ├── app.go                    # App struct, Deps struct, New() constructor
        ├── export.go                 # ExportService (THE hot path)
        ├── load.go                   # LoadService
        ├── clear.go                  # ClearService
        ├── list.go                   # ListService
        ├── sync.go                   # SyncService
        └── configure.go              # ConfigureService
```

## 4. Domain Layer (`internal/domain/`)

The domain layer contains pure business entities with no infrastructure dependencies.

### `project.go` - Project Entity

```go
type Project struct {
    ID        int
    Path      string
    Name      string
    CreatedAt string
}
```

Represents a registered project directory. The `Path` is always an absolute path.

### `session.go` - Session and SessionKey

```go
type Session struct {
    ShellPID     int
    ProjectPath  string
    EnvFileMtime int64
    LoadedAt     string
}

type SessionKey struct {
    ShellPID int
    KeyName  string
    KeyHash  string
}
```

- **Session**: Tracks which project is loaded in which shell session (by parent PID)
- **SessionKey**: Stores hashes of loaded environment variables for change detection
- **KeyNames()**: Helper to extract key names from `[]SessionKey`

### `envfile.go` - EnvFile and Diff Logic

```go
type EnvFile struct {
    Path   string
    Mtime  int64
    Values map[string]string
}

type DiffResult struct {
    Export map[string]string
    Unset  []string
}
```

#### Key Functions

**`HashValue(value string) string`**
- Uses SHA-256 truncated to 8 bytes (16 hex chars)
- Allows change detection without storing secrets in the database
- Example: `"secret123"` → `"a1b2c3d4e5f6a7b8"`

**`KeyHashes(ef *EnvFile) map[string]string`**
- Converts an EnvFile's values to a map of `key → hash`
- Used to store session state

**`Diff(envFile *EnvFile, loadedKeys []SessionKey) DiffResult`**
- Compares new `.env` contents against previously loaded keys
- Returns which keys to **export** (new or changed) and which to **unset** (removed)
- Handles nil envFile (no .env file → unset everything)

**Key Design Decision**: SHA-256 truncated hashes provide change detection without exposing secrets in the sessions database.

### `errors.go` - Sentinel Errors and Types

```go
var (
    ErrProjectNotFound = errors.New("project not found")
    ErrSessionNotFound = errors.New("session not found")
    ErrNoEnvFile       = errors.New("no .env file found")
)

type DefaultSetting struct {
    Key   string
    Value string
}
```

## 5. Port Layer (`internal/port/`)

Defines interfaces for adapters. Follows **Interface Segregation Principle**.

### `repository.go` - Data Persistence Interfaces

```go
type ProjectReader interface {
    MatchCurrent(dir string) (*domain.Project, error)
    ListAll() ([]domain.Project, error)
    FindByPath(path string) (*domain.Project, error)
}

type ProjectWriter interface {
    Upsert(path, name string) error
    Delete(path string) error
}

type ProjectRepository interface {
    ProjectReader
    ProjectWriter
}

type SessionRepository interface {
    Get(shellPID int) (*domain.Session, error)
    Upsert(shellPID int, projectPath string, envFileMtime int64) error
    Delete(shellPID int) error
    GetKeys(shellPID int) ([]domain.SessionKey, error)
    SetKeys(shellPID int, keys map[string]string) error
}
```

**Interface Segregation**: Splitting `ProjectReader` and `ProjectWriter` allows services to depend only on what they need (e.g., read-only operations don't need write methods).

**MatchCurrent()**: The most important method - finds the project that matches the current directory using longest path prefix matching.

### `envloader.go` - Environment File Loading

```go
type EnvLoader interface {
    Load(projectPath string) (*domain.EnvFile, error)
}
```

Returns `nil` (not an error) when no `.env` file exists.

### `shellrenderer.go` - Shell Command Formatting

```go
type ShellRenderer interface {
    FormatExports(shellType string, vars map[string]string) string
    FormatUnsets(shellType string, keys []string) string
}
```

Formats shell commands for exporting/unsetting variables.

### `configstore.go` - Configuration Storage

```go
type ConfigStore interface {
    Get(key string) (string, error)
    Set(key, value string) error
    List() ([]domain.DefaultSetting, error)
}
```

Key-value store for defaults (e.g., `github.default_owner`).

### `secretsync.go` - Secret Synchronization

```go
type SecretSyncer interface {
    Sync(repo string, secrets map[string]string) error
}
```

Syncs secrets to external targets (currently GitHub via `gh` CLI).

## 6. Adapter Layer (`internal/adapter/`)

Implements the port interfaces with concrete infrastructure.

### SQLite Adapter (`adapter/sqlite/`)

#### `turso.go` - Turso Embedded Replica

```go
type TursoDB struct {
    DB        *sql.DB
    connector *libsql.Connector
}

func OpenTurso(dbPath, tursoURL, authToken string) (*TursoDB, error)
func (t *TursoDB) Sync() error
func (t *TursoDB) Close() error
```

**Key Design**: Takes URL and auth token directly (decoupled from config). Falls back to local-only SQLite if credentials aren't provided.

**Embedded Replica Pattern**: Uses `libsql.NewEmbeddedReplicaConnector` with:
- Local SQLite file for fast reads/writes
- Background sync to Turso cloud
- `SyncInterval: 0` (manual sync only)

#### `local.go` - Local SQLite with WAL Mode

```go
func OpenLocal(path string) (*sql.DB, error)
func OpenSessionsDB(path string) (*sql.DB, io.Closer, error)
```

**WAL Mode**: Enables Write-Ahead Logging for better concurrency
- `PRAGMA journal_mode=WAL`
- Safer for multiple readers
- Better performance

**libsql Quirks**:
- Foreign keys pragma may not be supported (error ignored)
- Requires `SetMaxOpenConns(1)` for embedded replica mode

#### `migrations.go` - Schema Creation

**Critical libsql Limitation**: Must use separate `Exec()` calls per `CREATE TABLE` statement. Combined statements in a single `Exec()` will fail.

```go
func migrateProjects(db *sql.DB) error
func migrateSessions(db *sql.DB) error
```

See section 10 for full schema details.

#### `project_repo.go` - Project Repository

```go
type ProjectRepo struct {
    db *sql.DB
}

func NewProjectRepo(db *sql.DB) *ProjectRepo
```

**Compile-time interface check**:
```go
var (
    _ port.ProjectReader = (*ProjectRepo)(nil)
    _ port.ProjectWriter = (*ProjectRepo)(nil)
)
```

**MatchCurrent() Implementation**:
- Fetches all projects ordered by path length (longest first)
- Uses `filepath.Rel()` to check if current dir is within project path
- Returns first match (longest prefix wins)
- Example: `/home/user/project/subdir` matches `/home/user/project`, not `/home/user`

```go
func (r *ProjectRepo) MatchCurrent(dir string) (*domain.Project, error) {
    abs, err := filepath.Abs(dir)
    if err != nil {
        return nil, fmt.Errorf("resolve path: %w", err)
    }

    rows, err := r.db.Query(
        `SELECT id, path, name, created_at FROM projects ORDER BY length(path) DESC`,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var p domain.Project
        if err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.CreatedAt); err != nil {
            return nil, err
        }
        rel, err := filepath.Rel(p.Path, abs)
        if err != nil {
            continue
        }
        // If relative path starts with "..", we're outside the project
        if len(rel) >= 2 && rel[:2] == ".." {
            continue
        }
        return &p, nil
    }
    return nil, rows.Err()
}
```

#### `session_repo.go` - Session Repository

```go
type SessionRepo struct {
    db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo
```

**SetKeys() - Transaction-Based Atomic Update**:
1. Begin transaction
2. Delete all existing keys for the shell PID
3. Insert new keys
4. Commit

This ensures session keys are always consistent.

```go
func (r *SessionRepo) SetKeys(shellPID int, keys map[string]string) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    _, err = tx.Exec(`DELETE FROM session_keys WHERE shell_pid = ?`, shellPID)
    if err != nil {
        return err
    }

    stmt, err := tx.Prepare(
        `INSERT INTO session_keys (shell_pid, key_name, key_hash) VALUES (?, ?, ?)`,
    )
    if err != nil {
        return err
    }
    defer stmt.Close()

    for name, hash := range keys {
        if _, err := stmt.Exec(shellPID, name, hash); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

#### `defaults_repo.go` - Configuration Store

```go
type DefaultsRepo struct {
    db *sql.DB
}

func NewDefaultsRepo(db *sql.DB) *DefaultsRepo
```

Simple key-value store using `ON CONFLICT ... DO UPDATE` (upsert pattern).

```go
func (r *DefaultsRepo) Set(key, value string) error {
    _, err := r.db.Exec(
        `INSERT INTO defaults (key, value) VALUES (?, ?)
         ON CONFLICT(key) DO UPDATE SET value = excluded.value,
         updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
        key, value,
    )
    return err
}
```

### Shell Adapter (`adapter/shell/`)

#### `hook.go` - Shell Hook Scripts

```go
func HookScript(shellType string) (string, error)
```

Generates shell-specific hook code:

**Zsh Hook**: Uses `chpwd_functions` array
- Zsh calls functions in `chpwd_functions` whenever directory changes
- Appends `_autoenv_hook` to array if not already present
- Calls immediately to load current directory's .env

```bash
_autoenv_hook() {
  eval "$(autoenv export zsh)"
}
typeset -ag chpwd_functions
if [[ -z "${chpwd_functions[(r)_autoenv_hook]+1}" ]]; then
  chpwd_functions=(_autoenv_hook $chpwd_functions)
fi
_autoenv_hook
```

**Bash Hook**: Uses `PROMPT_COMMAND`
- Bash executes `PROMPT_COMMAND` before each prompt
- Preserves previous exit code with `prev_exit=$?` / `return $prev_exit`
- Appends to existing `PROMPT_COMMAND` if set

```bash
_autoenv_hook() {
  local prev_exit=$?
  eval "$(autoenv export bash)"
  return $prev_exit
}
if [[ ";${PROMPT_COMMAND[*]:-};" != *";_autoenv_hook;"* ]]; then
  PROMPT_COMMAND="_autoenv_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
fi
_autoenv_hook
```

#### `renderer.go` - Shell Command Formatter

```go
type Renderer struct{}

func NewRenderer() *Renderer
```

**Deterministic Output**: Keys are sorted alphabetically to ensure consistent output.

**Single-Quote Escaping**: Values are wrapped in single quotes with proper escaping:
- `'` → `'\''` (end quote, escaped quote, start quote)
- Example: `O'Reilly` → `export AUTHOR='O'\''Reilly'`

```go
func (r *Renderer) FormatExports(shellType string, vars map[string]string) string {
    if len(vars) == 0 {
        return ""
    }

    keys := make([]string, 0, len(vars))
    for k := range vars {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    var b strings.Builder
    for _, k := range keys {
        escaped := strings.ReplaceAll(vars[k], "'", "'\\''")
        fmt.Fprintf(&b, "export %s='%s'\n", k, escaped)
    }
    return b.String()
}
```

### EnvFile Adapter (`adapter/envfile/`)

#### `loader.go` - .env File Loader

```go
type Loader struct{}

func NewLoader() *Loader
```

Wraps `github.com/joho/godotenv` library. Captures file modification time for change detection.

```go
func (l *Loader) Load(projectPath string) (*domain.EnvFile, error) {
    envPath := filepath.Join(projectPath, ".env")

    info, err := os.Stat(envPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil  // No .env file is not an error
        }
        return nil, fmt.Errorf("stat %s: %w", envPath, err)
    }

    values, err := godotenv.Read(envPath)
    if err != nil {
        return nil, fmt.Errorf("parse %s: %w", envPath, err)
    }

    return &domain.EnvFile{
        Path:   envPath,
        Mtime:  info.ModTime().UnixNano(),
        Values: values,
    }, nil
}
```

### Config Adapter (`adapter/config/`)

#### `config.go` - XDG-Compliant Configuration

```go
type Config struct {
    Dir            string
    ProjectsDBPath string
    SessionsDBPath string
    TursoURL       string
    TursoAuthToken string
}

func Load() *Config
func (c *Config) EnsureDir() error
```

**Configuration Directory Resolution** (in order):
1. `AUTOENV_CONFIG_DIR` env var
2. `XDG_CONFIG_HOME/autoenv`
3. `~/.config/autoenv` (fallback)

**Database Paths**:
- `projects.db` - Turso-synced (projects + defaults)
- `sessions.db` - Local-only (sessions + session_keys)

### GitHub Adapter (`adapter/github/`)

#### `secrets.go` - GitHub Secrets Syncer

```go
type SecretSyncer struct{}

func NewSecretSyncer() *SecretSyncer
```

Shells out to `gh secret set` for each key-value pair. Requires `gh` CLI to be installed and authenticated.

```go
func (s *SecretSyncer) Sync(repo string, secrets map[string]string) error {
    if _, err := exec.LookPath("gh"); err != nil {
        return fmt.Errorf("gh CLI not found: install from https://cli.github.com")
    }

    for key, value := range secrets {
        cmd := exec.Command("gh", "secret", "set", key,
            "--repo", repo,
            "--body", value,
        )
        if out, err := cmd.CombinedOutput(); err != nil {
            return fmt.Errorf("set secret %s: %s: %w", key, strings.TrimSpace(string(out)), err)
        }
    }

    return nil
}
```

## 7. Application Layer (`internal/app/`)

Orchestrates business logic by composing ports (interfaces).

### `app.go` - App Container and Dependency Injection

```go
type App struct {
    Export    *ExportService
    Load      *LoadService
    Clear     *ClearService
    List      *ListService
    Sync      *SyncService
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

func New(d Deps) *App
```

**Dependency Injection**: All adapters are passed in via `Deps`, making services testable with mocks.

### `export.go` - ExportService (THE HOT PATH)

This is called on **every directory change** by the shell hook. Performance matters.

```go
type ExportService struct {
    projects  port.ProjectReader
    sessions  port.SessionRepository
    envLoader port.EnvLoader
    shell     port.ShellRenderer
}

func (s *ExportService) Export(shellType string, shellPID int, cwd string) (string, error)
```

#### Export Flow (Step-by-Step)

**1. Find Project**
```go
project, err := s.projects.MatchCurrent(cwd)
```
Uses longest path prefix matching. Returns `nil` if not in a registered project.

**2. Get Session State**
```go
session, err := s.sessions.Get(shellPID)
loadedKeys, err := s.sessions.GetKeys(shellPID)
```
Retrieves current session and loaded keys.

**3. Not in a Project → Cleanup**
```go
if project == nil {
    if session == nil {
        return "", nil  // No-op
    }
    output := s.shell.FormatUnsets(shellType, domain.KeyNames(loadedKeys))
    _ = s.sessions.Delete(shellPID)
    return output, nil
}
```
If we left a project, unset all loaded vars and delete the session.

**4. Load .env File**
```go
envFile, err := s.envLoader.Load(project.Path)
```
Returns `nil` if no `.env` file exists (not an error).

**5. Mtime Check for No-Op**
```go
if session != nil && session.ProjectPath == project.Path &&
   envFile != nil && session.EnvFileMtime == envFile.Mtime {
    return "", nil  // Same project, unchanged .env
}
```
**Performance optimization**: Skip diff if still in same project with unchanged .env.

**6. Compute Diff**
```go
diff := domain.Diff(envFile, loadedKeys)
```
Determines which keys to export (new/changed) and unset (removed).

**7. Handle Project Switching**
```go
if session != nil && session.ProjectPath != project.Path {
    for _, k := range loadedKeys {
        if envFile == nil {
            diff.Unset = append(diff.Unset, k.KeyName)
        } else if _, exists := envFile.Values[k.KeyName]; !exists {
            diff.Unset = append(diff.Unset, k.KeyName)
        }
    }
}
```
When switching projects, ensure old project's keys are unset if they don't exist in new project.

**8. Render Shell Commands**
```go
output := s.shell.FormatUnsets(shellType, diff.Unset) +
          s.shell.FormatExports(shellType, diff.Export)
```

**9. Update Session State**
```go
if envFile != nil {
    _ = s.sessions.Upsert(shellPID, project.Path, envFile.Mtime)
    _ = s.sessions.SetKeys(shellPID, domain.KeyHashes(envFile))
} else {
    _ = s.sessions.Delete(shellPID)
}
```

Store hashes of loaded keys for future diff operations.

### `load.go` - LoadService

```go
type LoadService struct {
    projects  port.ProjectRepository
    sessions  port.SessionRepository
    envLoader port.EnvLoader
    shell     port.ShellRenderer
}

func (s *LoadService) LoadProject(shellType string, shellPID int, projectPath, name string) (string, error)
```

Registers a project and immediately loads its .env:
1. Upsert project into database
2. Load .env file
3. Format export commands
4. Update session state

### `clear.go` - ClearService

```go
type ClearService struct {
    sessions port.SessionRepository
    shell    port.ShellRenderer
}

func (s *ClearService) Clear(shellType string, shellPID int) (string, error)
```

**No Project Dependency**: Only needs `SessionRepository` + `ShellRenderer`. This is why `cmd/clear.go` uses `bootstrapLight()` - no need to connect to Turso.

### `list.go` - ListService

```go
type ListService struct {
    projects port.ProjectReader
}

func (s *ListService) ListProjects() ([]domain.Project, error)
```

Thin wrapper over `ProjectReader.ListAll()`.

### `sync.go` - SyncService

```go
type SyncService struct {
    projects  port.ProjectReader
    envLoader port.EnvLoader
    syncer    port.SecretSyncer
    config    port.ConfigStore
}

func (s *SyncService) SyncSecrets(projectPath, target string) error
```

**Target Resolution**:
1. Strips `github.com/` prefix if present
2. If target contains `/`, it's already `owner/repo`
3. If bare repo name, prepends `github.default_owner` from config

```go
func (s *SyncService) resolveTarget(target string) (string, error) {
    // "github.com/owner/repo" → "owner/repo"
    target = strings.TrimPrefix(target, "github.com/")

    // Already "owner/repo"
    if strings.Contains(target, "/") {
        return target, nil
    }

    // Bare "repo" → "default_owner/repo"
    owner, err := s.config.Get("github.default_owner")
    if err != nil {
        return "", fmt.Errorf("no default owner configured; run: autoenv configure set github.default_owner <owner>")
    }

    return owner + "/" + target, nil
}
```

### `configure.go` - ConfigureService

```go
type ConfigureService struct {
    config port.ConfigStore
}

func (s *ConfigureService) Set(key, value string) error
func (s *ConfigureService) Get(key string) (string, error)
func (s *ConfigureService) List() ([]domain.DefaultSetting, error)
```

Thin wrapper over `ConfigStore` interface.

## 8. Command Layer (`cmd/`)

Wires adapters and services together, handles CLI concerns.

### `root.go` - Root Command

```go
var Version = "dev"  // Set by main.go via ldflags

var rootCmd = &cobra.Command{
    Use:   "autoenv",
    Short: "Automatically load .env files into shell sessions",
    Long:  "Autoenv loads .env files when you enter registered project directories and unsets them when you leave.",
}

func Execute()
```

### `bootstrap.go` - Dependency Injection Bootstrapping

Two bootstrap functions for different dependency profiles:

#### `bootstrap()` - Full Bootstrap

Used by: `export`, `load`, `list`, `sync`, `configure`

Opens:
- Turso DB (projects + defaults)
- Sessions DB
- All adapters

```go
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
```

#### `bootstrapLight()` - Light Bootstrap

Used by: `clear`

Opens only:
- Sessions DB
- Shell renderer
- EnvLoader (for minimal deps)

**Why?** ClearService only needs session state, not projects. Avoids unnecessary Turso connection for a simple operation.

```go
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
```

#### Closers Pattern

```go
type closers []io.Closer

func (c *closers) Add(closer io.Closer) { *c = append(*c, closer) }
func (c closers) CloseAll() {
    for i := len(c) - 1; i >= 0; i-- {
        _ = c[i].Close()
    }
}
```

Ensures all resources are closed in reverse order (LIFO).

### Command Files

Each command file follows this pattern:
1. Define cobra command
2. Call appropriate bootstrap function
3. Defer cleanup with `cc.CloseAll()`
4. Call service method
5. Handle output/errors
6. Register command in `init()`

#### `export.go` - Export Command

```go
var exportCmd = &cobra.Command{
    Use:    "export <shell>",
    Short:  "Emit export/unset commands (called by shell hook)",
    Args:   cobra.ExactArgs(1),
    Hidden: true,
    Run: func(cmd *cobra.Command, args []string) {
        b, err := bootstrap()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            return
        }
        defer b.cc.CloseAll()

        cwd, err := os.Getwd()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            return
        }

        output, err := b.app.Export.Export(args[0], getShellPID(), cwd)
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            return
        }
        fmt.Print(output)
    },
}

func getShellPID() int {
    pid, _ := strconv.Atoi(os.Getenv("AUTOENV_SHELL_PID"))
    if pid != 0 {
        return pid
    }
    return os.Getppid()
}
```

**Shell PID Detection**: Uses `AUTOENV_SHELL_PID` env var if set, otherwise parent PID.

#### `load.go` - Load Command

```go
var loadProject string

var loadCmd = &cobra.Command{
    Use:   "load",
    Short: "Register a project and output export commands for its .env",
    Long:  `Register a project directory and output export commands. Use with eval: eval "$(autoenv load --project /path/to/project)"`,
    Run: func(cmd *cobra.Command, args []string) {
        b, err := bootstrap()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }
        defer b.cc.CloseAll()

        projectPath := loadProject
        if projectPath == "" {
            projectPath, _ = os.Getwd()
        }

        absPath, err := filepath.Abs(projectPath)
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }

        name := filepath.Base(absPath)

        output, err := b.app.Load.LoadProject("zsh", getShellPID(), absPath, name)
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "autoenv: registered project %q (%s)\n", name, absPath)
        fmt.Print(output)
    },
}
```

#### `clear.go` - Clear Command

```go
var clearCmd = &cobra.Command{
    Use:   "clear",
    Short: "Unset all autoenv-loaded vars for current session",
    Long:  `Unset all autoenv-loaded environment variables. Use with eval: eval "$(autoenv clear)"`,
    Run: func(cmd *cobra.Command, args []string) {
        a, cc, err := bootstrapLight()  // ← Uses light bootstrap
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }
        defer cc.CloseAll()

        output, err := a.Clear.Clear("zsh", getShellPID())
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }

        fmt.Print(output)
    },
}
```

#### `list.go` - List Command

```go
var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List registered projects",
    Run: func(cmd *cobra.Command, args []string) {
        b, err := bootstrap()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }
        defer b.cc.CloseAll()

        projects, err := b.app.List.ListProjects()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }

        if len(projects) == 0 {
            fmt.Println("No registered projects.")
            return
        }

        w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
        fmt.Fprintln(w, "NAME\tPATH\tCREATED")
        for _, p := range projects {
            name := p.Name
            if name == "" {
                name = "-"
            }
            fmt.Fprintf(w, "%s\t%s\t%s\n", name, p.Path, p.CreatedAt)
        }
        w.Flush()
    },
}
```

#### `sync.go` - Sync Command

```go
var syncDB bool

var syncCmd = &cobra.Command{
    Use:   "sync [target]",
    Short: "Sync secrets to external targets or force Turso DB sync",
    Long: `Sync .env secrets to external targets like GitHub Actions.

Examples:
  autoenv sync github.com/stormingluke/stormingplatform   # full target
  autoenv sync stormingplatform                            # uses default owner
  autoenv sync --db                                        # Turso cloud sync`,
    Run: func(cmd *cobra.Command, args []string) {
        b, err := bootstrap()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }
        defer b.cc.CloseAll()

        if syncDB {
            if err := b.turso.Sync(); err != nil {
                fmt.Fprintf(os.Stderr, "autoenv: sync failed: %v\n", err)
                os.Exit(1)
            }
            fmt.Println("Turso sync complete.")
            return
        }

        if len(args) == 0 {
            fmt.Fprintln(os.Stderr, "autoenv: target required (e.g., autoenv sync github.com/owner/repo) or use --db for Turso sync")
            os.Exit(1)
        }

        cwd, err := os.Getwd()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }

        if err := b.app.Sync.SyncSecrets(cwd, args[0]); err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: sync failed: %v\n", err)
            os.Exit(1)
        }

        fmt.Printf("Secrets synced to %s\n", args[0])
    },
}
```

**Dual Purpose**:
- `--db` flag: Force Turso cloud sync
- Otherwise: Sync secrets to GitHub repo

#### `configure.go` - Configure Command

```go
var configureCmd = &cobra.Command{
    Use:   "configure",
    Short: "Manage autoenv defaults",
    Long: `Manage autoenv defaults that are stored in Turso and synced across machines.

Examples:
  autoenv configure set github.default_owner stormingluke
  autoenv configure get github.default_owner
  autoenv configure list`,
}

var configureSetCmd = &cobra.Command{
    Use:   "set <key> <value>",
    Short: "Set a default value",
    Args:  cobra.ExactArgs(2),
    Run: func(cmd *cobra.Command, args []string) {
        b, err := bootstrap()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }
        defer b.cc.CloseAll()

        if err := b.app.Configure.Set(args[0], args[1]); err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("Set %s = %s\n", args[0], args[1])
    },
}

// ... configureGetCmd and configureListCmd similar ...

func init() {
    configureCmd.AddCommand(configureSetCmd)
    configureCmd.AddCommand(configureGetCmd)
    configureCmd.AddCommand(configureListCmd)
    rootCmd.AddCommand(configureCmd)
}
```

#### `hook.go` - Hook Command

```go
var hookCmd = &cobra.Command{
    Use:   "hook <shell>",
    Short: "Output shell hook code to add to your shell config",
    Long:  `Output shell hook code. Add eval "$(autoenv hook zsh)" to your .zshrc.`,
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        script, err := shell.HookScript(args[0])
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        fmt.Print(script)
    },
}
```

**No Bootstrap**: This command doesn't need any database access, just returns static shell code.

## 9. CI/CD Pipeline

### `.dagger/main.go` - Dagger CI Pipeline

Dagger provides containerized, reproducible CI that runs the same locally and in GitHub Actions.

```go
type Autoenv struct{}
```

#### Functions

**`Lint(ctx context.Context, source *dagger.Directory) (string, error)`**
- Uses `golang:1.25-bookworm` base image
- Installs `golangci-lint`
- Runs `golangci-lint run ./...`

**`Test(ctx context.Context, source *dagger.Directory) (string, error)`**
- Uses `golang:1.25-bookworm` base image
- Runs `go test -race -count=1 ./...`
- Race detection enabled

**`Build(ctx, source, goos, goarch) *dagger.File`**
- Parameterized build for any GOOS/GOARCH
- Default: `linux/amd64`
- For `darwin/arm64`: Uses zig for cross-compilation
- Returns binary as Dagger file artifact

**`BuildAll(ctx, source) *dagger.Directory`**
- Builds both `linux/amd64` and `darwin/arm64`
- Returns directory with both binaries

**`Release(ctx, source, githubToken, snapshot) (string, error)`**
- Uses `golang:1.25-bookworm` + zig
- Installs `goreleaser`
- Runs `goreleaser release --clean`
- With `--snapshot` for local testing

**`All(ctx, source) (string, error)`**
- Runs lint, test, and buildAll in sequence
- Used by CI job

#### Helper Functions

**`base(source) *dagger.Container`**
- Returns Go build container without zig
- Used for lint/test/native builds
- Sets `CGO_ENABLED=1`

**`withZig(ctr) *dagger.Container`**
- Downloads zig 0.14.0 for darwin cross-compilation
- Installs to `/opt/zig-linux-x86_64-0.14.0/`
- Symlinks to `/usr/local/bin/zig`

```go
func (m *Autoenv) withZig(ctr *dagger.Container) *dagger.Container {
    zigURL := fmt.Sprintf("https://ziglang.org/download/%s/zig-linux-x86_64-%s.tar.xz", zigVersion, zigVersion)
    return ctr.
        WithExec([]string{"sh", "-c",
            fmt.Sprintf("curl -fsSL %s | tar xJ -C /opt && ln -sf /opt/zig-linux-x86_64-%s/zig /usr/local/bin/zig", zigURL, zigVersion)})
}
```

### `.github/workflows/ci.yml` - GitHub Actions

```yaml
jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: CI (lint + test + build)
        uses: dagger/dagger-for-github@v8.2.0
        with:
          version: latest
          call: all --source=.

  release:
    needs: ci
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Release
        uses: dagger/dagger-for-github@v8.2.0
        with:
          version: latest
          call: release --source=. --github-token=env:GITHUB_TOKEN
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Two Jobs**:
1. **ci**: Runs on every push/PR - lints, tests, builds
2. **release**: Runs only on version tags (`v*`) - creates GitHub release

### `.goreleaser.yaml` - Release Configuration

```yaml
builds:
  - id: autoenv-darwin
    binary: autoenv
    goos: [darwin]
    goarch: [arm64]
    env:
      - CGO_ENABLED=1
      - CC=zig cc -target aarch64-macos
      - CXX=zig c++ -target aarch64-macos
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

  - id: autoenv-linux
    binary: autoenv
    goos: [linux]
    goarch: [amd64]
    env:
      - CGO_ENABLED=1
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
```

**Two Builds**:
- `autoenv-darwin`: macOS ARM64 with zig cross-compilation
- `autoenv-linux`: Linux AMD64 native

**ldflags**: Injects version/commit/date into `main.go` variables.

### `Taskfile.yml` - Local Development Tasks

```yaml
tasks:
  default:
    desc: Run CI pipeline (lint, test, build)
    aliases: [ci]
    cmds:
      - task: lint
      - task: test
      - task: build

  build:
    desc: Build the autoenv binary
    cmds:
      - go build -trimpath -o {{.BINARY}} .
    sources:
      - '**/*.go'
      - go.mod
      - go.sum
    generates:
      - '{{.BINARY}}'

  test:
    desc: Run tests with race detection
    cmds:
      - go test -race -count=1 ./...

  lint:
    desc: Run golangci-lint
    cmds:
      - golangci-lint run ./...

  install:
    desc: Build and install to GOPATH
    deps: [build]
    cmds:
      - cp {{.BINARY}} $(go env GOPATH)/bin/{{.BINARY}}

  dagger:
    desc: Run full Dagger CI pipeline
    cmds:
      - dagger call all --source=.
```

**Common Commands**:
- `task` or `task ci` - Run full CI locally
- `task build` - Build binary
- `task test` - Run tests
- `task lint` - Run linter
- `task install` - Install to GOPATH
- `task dagger` - Run Dagger pipeline locally

## 10. Database Schema

### Projects Database (`projects.db`) - Turso-Synced

#### `projects` Table

```sql
CREATE TABLE IF NOT EXISTS projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    path       TEXT NOT NULL UNIQUE,
    name       TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
)
```

- `id`: Auto-incrementing primary key
- `path`: Absolute path to project directory (UNIQUE)
- `name`: Project name (derived from directory basename)
- `created_at`: ISO 8601 timestamp

#### `defaults` Table

```sql
CREATE TABLE IF NOT EXISTS defaults (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
)
```

- `key`: Configuration key (e.g., `github.default_owner`)
- `value`: Configuration value
- `updated_at`: ISO 8601 timestamp

**Use Case**: Store user preferences that sync across machines via Turso.

### Sessions Database (`sessions.db`) - Local-Only

#### `sessions` Table

```sql
CREATE TABLE IF NOT EXISTS sessions (
    shell_pid      INTEGER PRIMARY KEY,
    project_path   TEXT NOT NULL,
    env_file_mtime INTEGER NOT NULL,
    loaded_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
)
```

- `shell_pid`: Parent shell process ID (primary key)
- `project_path`: Absolute path to currently loaded project
- `env_file_mtime`: Modification time of .env file (nanoseconds since epoch)
- `loaded_at`: ISO 8601 timestamp

**Mtime Purpose**: Allows fast change detection without reading file contents.

#### `session_keys` Table

```sql
CREATE TABLE IF NOT EXISTS session_keys (
    shell_pid  INTEGER NOT NULL REFERENCES sessions(shell_pid) ON DELETE CASCADE,
    key_name   TEXT NOT NULL,
    key_hash   TEXT NOT NULL,
    PRIMARY KEY (shell_pid, key_name)
)
```

- `shell_pid`: Foreign key to `sessions` table
- `key_name`: Environment variable name
- `key_hash`: SHA-256 truncated hash of value (16 hex chars)
- `ON DELETE CASCADE`: Automatically clean up keys when session is deleted

**Hash Purpose**: Enables change detection without storing actual secret values.

## 11. Adding a New Command

Follow these steps to extend autoenv with a new command:

### Step 1: Define Domain Types (if needed)

If your command introduces new concepts, add entities to `internal/domain/`.

Example: Adding a project template feature
```go
// internal/domain/template.go
package domain

type ProjectTemplate struct {
    Name        string
    Description string
    EnvTemplate map[string]string
}
```

### Step 2: Add Port Interface (if needed)

Define the interface your service will depend on in `internal/port/`.

```go
// internal/port/templatestore.go
package port

import "github.com/stormingluke/autoenv/internal/domain"

type TemplateStore interface {
    List() ([]domain.ProjectTemplate, error)
    Get(name string) (*domain.ProjectTemplate, error)
    Save(template domain.ProjectTemplate) error
    Delete(name string) error
}
```

### Step 3: Implement Adapter (if needed)

Create the concrete implementation in `internal/adapter/`.

```go
// internal/adapter/sqlite/template_repo.go
package sqlite

import (
    "database/sql"
    "encoding/json"

    "github.com/stormingluke/autoenv/internal/domain"
    "github.com/stormingluke/autoenv/internal/port"
)

var _ port.TemplateStore = (*TemplateRepo)(nil)

type TemplateRepo struct {
    db *sql.DB
}

func NewTemplateRepo(db *sql.DB) *TemplateRepo {
    return &TemplateRepo{db: db}
}

func (r *TemplateRepo) List() ([]domain.ProjectTemplate, error) {
    // Implementation...
}

// ... other methods ...
```

Don't forget to add migration in `migrations.go`:
```go
func migrateTemplates(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS templates (
            name        TEXT PRIMARY KEY,
            description TEXT,
            env_template TEXT NOT NULL
        )
    `)
    return err
}
```

### Step 4: Create Service in `internal/app/`

Add service logic that orchestrates the domain and ports.

```go
// internal/app/template.go
package app

import (
    "github.com/stormingluke/autoenv/internal/domain"
    "github.com/stormingluke/autoenv/internal/port"
)

type TemplateService struct {
    templates port.TemplateStore
}

func (s *TemplateService) CreateFromTemplate(name string, projectPath string) error {
    template, err := s.templates.Get(name)
    if err != nil {
        return err
    }

    // Business logic here...

    return nil
}
```

Update `app.go`:
```go
type App struct {
    Export    *ExportService
    Load      *LoadService
    Clear     *ClearService
    List      *ListService
    Sync      *SyncService
    Configure *ConfigureService
    Template  *TemplateService  // ← Add this
}

type Deps struct {
    Projects  port.ProjectRepository
    Sessions  port.SessionRepository
    EnvLoader port.EnvLoader
    Shell     port.ShellRenderer
    Syncer    port.SecretSyncer
    Config    port.ConfigStore
    Templates port.TemplateStore  // ← Add this
}

func New(d Deps) *App {
    return &App{
        Export:    &ExportService{...},
        Load:      &LoadService{...},
        Clear:     &ClearService{...},
        List:      &ListService{...},
        Sync:      &SyncService{...},
        Configure: &ConfigureService{...},
        Template:  &TemplateService{templates: d.Templates},  // ← Add this
    }
}
```

### Step 5: Wire in `cmd/bootstrap.go`

Add adapter initialization in the bootstrap function:

```go
func bootstrap() (*bootstrapResult, error) {
    // ... existing code ...

    projectRepo := sqlite.NewProjectRepo(turso.DB)
    sessionRepo := sqlite.NewSessionRepo(sessDB)
    defaultsRepo := sqlite.NewDefaultsRepo(turso.DB)
    templateRepo := sqlite.NewTemplateRepo(turso.DB)  // ← Add this

    a := app.New(app.Deps{
        Projects:  projectRepo,
        Sessions:  sessionRepo,
        EnvLoader: envfile.NewLoader(),
        Shell:     shell.NewRenderer(),
        Syncer:    github.NewSecretSyncer(),
        Config:    defaultsRepo,
        Templates: templateRepo,  // ← Add this
    })

    return &bootstrapResult{app: a, turso: turso, cc: cc}, nil
}
```

### Step 6: Create Command in `cmd/`

Create a new command file:

```go
// cmd/template.go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
    Use:   "template",
    Short: "Manage project templates",
}

var templateListCmd = &cobra.Command{
    Use:   "list",
    Short: "List available templates",
    Run: func(cmd *cobra.Command, args []string) {
        b, err := bootstrap()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }
        defer b.cc.CloseAll()

        templates, err := b.app.Template.List()
        if err != nil {
            fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
            os.Exit(1)
        }

        for _, t := range templates {
            fmt.Printf("%s - %s\n", t.Name, t.Description)
        }
    },
}

// ... more subcommands ...

func init() {
    templateCmd.AddCommand(templateListCmd)
    // ... add other subcommands ...
    rootCmd.AddCommand(templateCmd)
}
```

### Step 7: Register in `init()`

The `init()` function at the bottom of your command file automatically registers the command when the package is imported. Make sure `main.go` imports `cmd` package:

```go
// main.go
package main

import (
    "fmt"

    "github.com/stormingluke/autoenv/cmd"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    cmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
    cmd.Execute()  // ← This triggers all init() functions
}
```

### Testing Your Command

1. **Build**: `task build`
2. **Run**: `./autoenv template list`
3. **Test**: `task test`
4. **Lint**: `task lint`

### Best Practices

- **Follow the dependency rule**: Inner layers don't import outer layers
- **Add compile-time interface checks**: `var _ port.Interface = (*Implementation)(nil)`
- **Use descriptive error messages**: Wrap errors with context using `fmt.Errorf("context: %w", err)`
- **Handle nil gracefully**: Check for nil before dereferencing pointers
- **Close resources**: Use the `closers` pattern for cleanup
- **Write tests**: Add unit tests for services and integration tests for commands
- **Update docs**: Document your new command in README.md

---

**Questions?** File an issue or submit a PR!
