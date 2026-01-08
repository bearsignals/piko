# Milestone 1: Core Create Flow

**Priority:** Critical
**Depends on:** Nothing
**Unlocks:** M2, M3, M4, M5

## Goal

`piko create <name>` works end-to-end: creates a git worktree, generates compose override, starts containers, and records state.

## Success Criteria

```bash
$ cd ~/projects/myapp           # has docker-compose.yml
$ piko create feature-auth

✓ Created worktree at .piko/worktrees/feature-auth
✓ Generated docker-compose.piko.yml
✓ Started containers (piko-myapp-feature-auth)
✓ Environment ready

$ docker ps | grep piko-myapp-feature-auth
# Shows running containers with random host ports

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
    config/      # project detection
  ```

### 1.2 Project Detection
- [ ] Find project root (walk up looking for `.git/` + `docker-compose.yml`)
- [ ] Extract project name from directory
- [ ] Validate docker-compose.yml exists

### 1.3 SQLite State
- [ ] Initialize database at `.piko/state.db`
- [ ] Create schema (project, environments tables)
- [ ] Functions: `InsertEnvironment`, `GetEnvironment`, `ListEnvironments`

### 1.4 Git Worktree
- [ ] Create `.piko/worktrees/` directory
- [ ] Run `git worktree add .piko/worktrees/<name> -b <name>`
- [ ] Handle existing branch case (`--branch` flag)
- [ ] Validate worktree was created

### 1.5 Port Allocation
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

### 1.6 Compose Override Generation
- [ ] Run `docker compose config --format json` in worktree directory
- [ ] Parse JSON to extract services and their ports
- [ ] Generate `docker-compose.piko.yml`:
  - Set deterministic host ports (from 1.5)
  - Set network name to `piko-<project>-<name>`
  - Namespace volumes: `piko-<project>-<name>-<volume>`
- [ ] Write file to worktree directory

### 1.7 Start Containers
- [ ] Run from worktree directory:
  ```bash
  docker compose -p piko-<project>-<name> \
    -f docker-compose.yml \
    -f docker-compose.piko.yml \
    up -d
  ```
- [ ] Capture and display output
- [ ] Handle errors (compose failure, port conflicts)

### 1.8 Run Setup Script
- [ ] Load `.piko.yml` from project root (if exists)
- [ ] If `scripts.setup` is defined:
  - Export PIKO_* environment variables (see 1.10)
  - cd to worktree directory
  - Execute script via shell
- [ ] Common setup tasks (handled by user's script):
  - Symlink env files: `ln -s "$PIKO_ROOT/.env.local" .env.local`
  - Install dependencies: `go mod download`, `npm install`
  - Run migrations: `make migrate`
- [ ] Capture output, fail if script fails

### 1.9 Record State
- [ ] Insert environment into SQLite:
  - id (auto-increment, used for port allocation)
  - name, branch, path, docker_project, created_at
- [ ] tmux_session field left empty (M3)

### 1.10 Env Command
- [ ] `piko env <name>` — print all PIKO_* variables
- [ ] `piko env <name> --json` — JSON output
- [ ] Variables to export:
  ```bash
  PIKO_ROOT=/home/user/project
  PIKO_ENV_NAME=feature-auth
  PIKO_ENV_PATH=/home/user/project/.piko/worktrees/feature-auth
  PIKO_PROJECT=piko-project-feature-auth
  PIKO_BRANCH=feature-auth
  PIKO_DB_PORT=10132
  PIKO_APP_PORT=10180
  PIKO_JAEGER_PORT=10118
  ```
- [ ] Service names uppercased, hyphens → underscores (my-service → PIKO_MY_SERVICE_PORT)
- [ ] Discover ports via `docker compose -p <project> port <service> <port>`

## Non-Goals (Deferred)

- Tmux session creation (M3)
- Shared services (M6)
- Full configuration (M7) — only basic scripts.setup support here
- Destroying environments (M2)
- Run script (M4)

## Test Cases

1. **Happy path**: Create environment in valid project
2. **No compose file**: Error with helpful message
3. **Not a git repo**: Error with helpful message
4. **Name already exists**: Error, suggest `piko destroy` first
5. **Branch already exists**: Use existing branch (don't create new)
6. **Compose with multiple files**: Works (docker handles via config)
7. **With setup script**: Script runs after containers start
8. **Setup script fails**: Create fails, containers stopped
9. **No .piko.yml**: Works (scripts are optional)

## Definition of Done

- [ ] `piko create <name>` creates worktree
- [ ] `piko create <name>` generates override file
- [ ] `piko create <name>` starts containers
- [ ] `piko create <name>` runs setup script (if defined)
- [ ] `piko create <name>` records to SQLite
- [ ] Containers have isolated network
- [ ] Containers have deterministic host ports (based on worktree ID)
- [ ] Volumes are namespaced per worktree
- [ ] `piko env <name>` outputs PIKO_* variables
- [ ] `piko env <name> --json` outputs JSON format
- [ ] Ports are deterministic for lifetime of worktree (based on DB ID)
- [ ] Running twice with same name shows error
