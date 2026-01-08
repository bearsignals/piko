# Milestone 1: Init & Create

**Priority:** Critical
**Depends on:** Nothing
**Unlocks:** M2, M3, M4, M5

## Goal

`piko init` initializes a project for piko tracking. `piko create <name>` creates isolated worktree environments.

## Success Criteria

```bash
$ cd ~/projects/myapp           # has docker-compose.yml
$ piko init
✓ Detected docker-compose.yml
✓ Created .piko/state.db
✓ Project "myapp" initialized

$ piko create feature-auth
✓ Created worktree at .piko/worktrees/feature-auth
✓ Generated docker-compose.piko.yml
✓ Started containers (piko-myapp-feature-auth)
✓ Ran setup script
✓ Environment ready

$ piko env feature-auth
PIKO_ROOT=/home/user/myapp
PIKO_ENV_NAME=feature-auth
PIKO_DB_PORT=10132
...

$ ls .piko/
state.db
worktrees/feature-auth/
```

## Tasks

### 1.1 Project Structure
- [ ] Initialize Go module
- [ ] Set up CLI with Cobra
- [ ] Create package structure:
  ```
  cmd/piko/main.go
  internal/
    git/         # worktree operations
    docker/      # compose operations
    state/       # SQLite database
    config/      # config loading (.piko.yml)
  ```

### 1.2 Init Command
- [ ] `piko init` — initialize project for piko
- [ ] Validate current directory:
  - Is a git repo (has `.git/`)
  - Has `docker-compose.yml` (or `docker-compose.yaml`, `compose.yml`)
- [ ] Create `.piko/` directory
- [ ] Initialize SQLite database at `.piko/state.db`
- [ ] Insert project record:
  - name (from directory name)
  - root_path (absolute path)
  - compose_file (detected filename)
- [ ] Add `.piko/` to `.gitignore` if not already present
- [ ] Error if already initialized (suggest `piko status`)

### 1.3 SQLite Schema
- [ ] Create tables:
  ```sql
  CREATE TABLE project (
      id INTEGER PRIMARY KEY,
      name TEXT UNIQUE NOT NULL,
      root_path TEXT NOT NULL,
      compose_file TEXT DEFAULT 'docker-compose.yml',
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE environments (
      id INTEGER PRIMARY KEY,
      project_id INTEGER REFERENCES project(id) ON DELETE CASCADE,
      name TEXT NOT NULL,
      branch TEXT NOT NULL,
      path TEXT NOT NULL,
      docker_project TEXT NOT NULL,
      tmux_session TEXT,
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      UNIQUE(project_id, name)
  );
  ```
- [ ] Functions: `GetProject`, `InsertEnvironment`, `GetEnvironment`, `ListEnvironments`

### 1.4 Config Loading
- [ ] Load `.piko.yml` from project root (if exists)
- [ ] Parse YAML into config struct
- [ ] Used by create command for scripts.setup
- [ ] Missing file is not an error (scripts are optional)

### 1.5 Create Command
- [ ] `piko create <name>` — create new environment
- [ ] Validate project is initialized (`.piko/state.db` exists)
- [ ] Validate name not already used

### 1.6 Git Worktree
- [ ] Create `.piko/worktrees/` directory if needed
- [ ] Run `git worktree add .piko/worktrees/<name> -b <name>`
- [ ] Handle existing branch case (`--branch` flag to use existing)
- [ ] Validate worktree was created

### 1.7 Port Allocation
- [ ] Assign worktree ID (auto-increment from SQLite)
- [ ] Calculate base port: `10000 + (worktree_id * 100)`
- [ ] For each service port: `base + (original_port % 100)`
- [ ] Example (worktree_id=1, base=10100):
  ```
  db:5432      → 10132
  jaeger:4318  → 10118
  app:8080     → 10180
  riverui:8081 → 10181
  ```

### 1.8 Compose Override Generation
- [ ] Run `docker compose config --format json` in worktree directory
- [ ] Parse JSON to extract services and their ports
- [ ] Generate `docker-compose.piko.yml`:
  - Set deterministic host ports (from 1.7)
  - Set network name to `piko-<project>-<name>`
  - Namespace volumes: `piko-<project>-<name>-<volume>`
- [ ] Write file to worktree directory

### 1.9 Start Containers
- [ ] Run from worktree directory:
  ```bash
  docker compose -p piko-<project>-<name> \
    -f docker-compose.yml \
    -f docker-compose.piko.yml \
    up -d
  ```
- [ ] Capture and display output
- [ ] Handle errors (compose failure, port conflicts)

### 1.10 Run Setup Script
- [ ] If `scripts.setup` is defined in `.piko.yml`:
  - Export PIKO_* environment variables
  - cd to worktree directory
  - Execute script via shell
- [ ] Common setup tasks (handled by user's script):
  - Symlink env files: `ln -s "$PIKO_ROOT/.env.local" .env.local`
  - Install dependencies: `go mod download`, `npm install`
  - Run migrations: `make migrate`
- [ ] Capture output, fail create if script fails

### 1.11 Record State
- [ ] Insert environment into SQLite:
  - id (auto-increment, used for port allocation)
  - name, branch, path, docker_project, created_at
- [ ] tmux_session field left empty (M4)

### 1.12 Env Command
- [ ] `piko env <name>` — print all PIKO_* variables
- [ ] `piko env <name> --json` — JSON output
- [ ] Variables:
  ```bash
  PIKO_ROOT=/home/user/project
  PIKO_ENV_NAME=feature-auth
  PIKO_ENV_PATH=/home/user/project/.piko/worktrees/feature-auth
  PIKO_PROJECT=piko-project-feature-auth
  PIKO_BRANCH=feature-auth
  PIKO_DB_PORT=10132
  PIKO_APP_PORT=10180
  ```
- [ ] Service names uppercased, hyphens → underscores
- [ ] Discover ports via `docker compose -p <project> port <service> <port>`

## Non-Goals (Deferred)

- Tmux session creation (M4)
- Shared services (M7)
- Full configuration (M8) — only basic scripts.setup support here
- Destroying environments (M2)
- Run script (M5)

## Test Cases

1. **Init happy path**: Creates .piko directory and database
2. **Init no compose file**: Error with helpful message
3. **Init not a git repo**: Error with helpful message
4. **Init already initialized**: Error, suggest `piko status`
5. **Create happy path**: Creates worktree and starts containers
6. **Create not initialized**: Error, suggest `piko init`
7. **Create name exists**: Error, suggest `piko destroy` first
8. **Create branch exists**: Use existing branch (don't create new)
9. **Create with setup script**: Script runs after containers start
10. **Create setup script fails**: Create fails, containers stopped
11. **Create no .piko.yml**: Works (scripts are optional)

## Definition of Done

- [ ] `piko init` creates .piko directory
- [ ] `piko init` creates SQLite database with schema
- [ ] `piko init` records project
- [ ] `piko init` adds .piko to .gitignore
- [ ] `piko create <name>` requires init first
- [ ] `piko create <name>` creates worktree
- [ ] `piko create <name>` generates override file
- [ ] `piko create <name>` starts containers
- [ ] `piko create <name>` runs setup script (if defined)
- [ ] `piko create <name>` records to SQLite
- [ ] Containers have isolated network
- [ ] Containers have deterministic host ports
- [ ] `piko env <name>` outputs PIKO_* variables
- [ ] Running create twice with same name shows error
