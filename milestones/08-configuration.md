# Milestone 8: Configuration

**Priority:** Low
**Depends on:** M1-M4 (Init + Lifecycle + Server + Tmux)
**Unlocks:** Project customization, user preferences

## Goal

Optional configuration files for project-specific settings and user preferences. Scripts are the primary mechanism for project-specific behavior. Zero-config remains the default.

## Success Criteria

```yaml
# .piko.yml in project root
scripts:
  setup: |
    ln -s "$PIKO_ROOT/.env.local" .env.local
    go mod download
    make migrate

  run: |
    DATABASE_URL="postgres://user:pass@localhost:$PIKO_DB_PORT/mydb" \
    go run cmd/main.go

  destroy: |
    echo "Cleaned up $PIKO_ENV_NAME"

shared:
  - jaeger

# Custom shell commands for tmux windows (exec into containers)
shells:
  db: psql -U myuser -d mydb
  redis: redis-cli

ignore:
  - riverui

windows:
  - name: frontend
    command: npm run dev
    local: true
```

```yaml
# ~/.config/piko/config.yml
editor: cursor
session_prefix: "dev:"
```

## Tasks

### 7.1 Project Config (.piko.yml)
- [ ] Load from project root
- [ ] Parse YAML
- [ ] Schema:
  ```go
  type ProjectConfig struct {
      Scripts ScriptsConfig     `yaml:"scripts"`
      Shared  []string          `yaml:"shared"`
      Shells  map[string]string `yaml:"shells"`
      Ignore  []string          `yaml:"ignore"`
      Windows []WindowConfig    `yaml:"windows"`
  }

  type ScriptsConfig struct {
      Setup   string `yaml:"setup"`
      Run     string `yaml:"run"`
      Destroy string `yaml:"destroy"`
  }

  type WindowConfig struct {
      Name    string `yaml:"name"`
      Command string `yaml:"command"`
      Local   bool   `yaml:"local"`
  }
  ```

### 7.2 User Config (~/.config/piko/config.yml)
- [ ] Load from XDG config dir
- [ ] Parse YAML
- [ ] Schema:
  ```go
  type UserConfig struct {
      Editor        string         `yaml:"editor"`
      SessionPrefix string         `yaml:"session_prefix"`
      DefaultWindows []WindowConfig `yaml:"default_windows"`
  }
  ```

### 7.3 Config Loading
- [ ] Load user config first (defaults)
- [ ] Load project config (overrides)
- [ ] CLI flags override both
- [ ] Handle missing files gracefully (not an error)

### 7.4 Scripts Execution
- [ ] Scripts run via shell (`sh -c`)
- [ ] Working directory is worktree path
- [ ] PIKO_* environment variables exported
- [ ] Output streamed to terminal
- [ ] Exit code checked (setup/run fail on error, destroy warns)

### 7.5 Shells Configuration
- [ ] Read `shells` from config
- [ ] Map of service name → shell command
- [ ] Used for tmux windows (docker exec command)
- [ ] Default: `sh` if not specified
- [ ] Example: `db: psql -U postgres` runs `docker exec ... db psql -U postgres`

### 7.6 Shared Services Integration
- [ ] Read `shared` from config
- [ ] Pass to M6 (Shared Services) when implemented
- [ ] For now: just validate the field

### 7.7 Ignore Services
- [ ] Read `ignore` from config
- [ ] Don't create tmux windows for these
- [ ] Still run containers (just not in tmux)

### 7.8 Custom Windows
- [ ] Read `windows` from config
- [ ] Create additional tmux windows
- [ ] `local: true` = run on host, not in container
- [ ] Support variable substitution: `$PIKO_PROJECT`, `$PIKO_ENV_NAME`

### 7.9 Editor Configuration
- [ ] Read from user config
- [ ] Use for `piko edit` command
- [ ] Fallback: $EDITOR → cursor → code → vim

### 7.10 Session Prefix
- [ ] Read from user config
- [ ] Change `piko/` prefix to custom value
- [ ] Example: `dev:` → `dev:myapp:feature-auth`

## Config Resolution

```
┌─────────────────────────────────────────┐
│  1. Defaults (hardcoded)                │
│     editor: cursor                      │
│     session_prefix: piko/               │
└──────────────────┬──────────────────────┘
                   ▼
┌─────────────────────────────────────────┐
│  2. User Config                         │
│     ~/.config/piko/config.yml           │
│     (overrides defaults)                │
└──────────────────┬──────────────────────┘
                   ▼
┌─────────────────────────────────────────┐
│  3. Project Config                      │
│     .piko.yml                           │
│     (overrides user config)             │
└──────────────────┬──────────────────────┘
                   ▼
┌─────────────────────────────────────────┐
│  4. CLI Flags                           │
│     --editor vim                        │
│     (overrides everything)              │
└─────────────────────────────────────────┘
```

## PIKO_* Environment Variables

Available in all scripts:

| Variable | Example |
|----------|---------|
| `PIKO_ROOT` | `/home/user/myapp` |
| `PIKO_ENV_NAME` | `feature-auth` |
| `PIKO_ENV_PATH` | `/home/user/myapp/.piko/worktrees/feature-auth` |
| `PIKO_PROJECT` | `piko-myapp-feature-auth` |
| `PIKO_BRANCH` | `feature-auth` |
| `PIKO_<SERVICE>_PORT` | `PIKO_DB_PORT=10132` |

## Test Cases

1. **No config files**: Uses defaults
2. **User config only**: Applies user preferences
3. **Project config only**: Applies project settings
4. **Both configs**: Merges correctly
5. **CLI override**: Takes precedence
6. **Invalid YAML**: Error with line number
7. **Unknown fields**: Ignore (forward compat)
8. **Custom windows**: Creates them
9. **Ignore service**: No tmux window
10. **Custom shell**: Uses specified command for tmux window
11. **Setup script**: Runs on create
12. **Run script**: Runs on `piko run`
13. **Destroy script**: Runs on destroy

## Definition of Done

- [ ] `.piko.yml` loaded from project root
- [ ] `~/.config/piko/config.yml` loaded
- [ ] Configs merge correctly
- [ ] `scripts.setup` runs after containers start
- [ ] `scripts.run` runs via `piko run`
- [ ] `scripts.destroy` runs before cleanup
- [ ] PIKO_* vars available in all scripts
- [ ] `shells` customizes tmux window exec commands
- [ ] `shared` marks services (for M6)
- [ ] `ignore` skips tmux windows
- [ ] `windows` adds custom windows
- [ ] `editor` configures edit command
- [ ] `session_prefix` changes naming
- [ ] Missing configs don't error
