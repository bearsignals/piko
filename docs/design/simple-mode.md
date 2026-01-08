# Simple Mode: Non-Docker Project Support

## Problem

Piko's core value is **worktrees + tmux** for isolated development environments. Currently it's tightly coupled to Docker Compose, but many projects (CLI tools, libraries, simple services) don't need containers.

## Design Principles

1. **Primitives over prescriptions** - Provide building blocks, let users compose
2. **Zero config by default** - Just works, configure only if needed
3. **Graceful degradation** - Docker mode and simple mode share the same core

## Core Primitives

### 1. Environment Variables

Piko provides `PIKO_*` variables as the primary mechanism for isolation:

```bash
# Identity
PIKO_PROJECT=myapp
PIKO_ENV=feature-xyz
PIKO_ENV_ID=42                       # Unique integer, useful for deriving ports

# Paths
PIKO_ROOT=/path/to/project           # Git root
PIKO_PATH=/path/to/worktree          # This worktree
PIKO_DATA_DIR=/path/to/.piko/data/feature-xyz  # Env-specific data
```

**User composes as needed:**

```javascript
// Derive unique port from env ID
server: { port: 3000 + parseInt(process.env.PIKO_ENV_ID || 0) }
```

```bash
# Use isolated data directory
DATABASE_URL=sqlite:${PIKO_DATA_DIR}/dev.db
```

### 2. Data Directory

Each environment gets an isolated data directory:

```
.piko/
├── worktrees/
│   ├── feature-a/        # git worktree
│   └── feature-b/        # git worktree
└── data/
    ├── feature-a/        # data for feature-a
    │   ├── dev.db
    │   └── cache/
    └── feature-b/        # data for feature-b
        ├── dev.db
        └── cache/
```

Created automatically on `piko create`. Cleaned up on `piko destroy`.

### 3. Scripts (Already Exists)

```yaml
scripts:
  prepare: "npm install"           # Before anything, in worktree
  setup: "npm run db:migrate"      # After services ready
  run: "npm run dev"               # Main dev command
  destroy: "rm -rf dist"           # Cleanup
```

### 4. Tmux Windows (Already Exists)

```yaml
shells:
  app: "npm run dev -- --port $((3000 + PIKO_ENV_ID))"
  worker: "npm run worker"
  logs: "tail -f $PIKO_DATA_DIR/app.log"
```

## Mode Detection

```
Has docker-compose.yml?
  ├── Yes → Docker mode (current behavior)
  └── No → Simple mode
```

## Command Behavior by Mode

| Command | Docker Mode | Simple Mode |
|---------|-------------|-------------|
| `create` | worktree + compose + tmux | worktree + tmux + data dir |
| `up` | docker compose up | no-op |
| `down` | docker compose down | no-op |
| `destroy` | compose down + rm worktree + data | rm worktree + data dir |
| `list` | show container status | show "simple" or tmux status |
| `status` | container details | env info only |
| `logs` | docker compose logs | no-op (user uses tmux) |
| `exec` | docker compose exec | no-op |
| `attach` | tmux attach | tmux attach |
| `env` | print PIKO_* vars | print PIKO_* vars |

## Isolation Strategies (User's Responsibility)

Piko provides primitives. Users apply them as needed:

### Ports

```javascript
// vite.config.js - derive port from env ID
server: { port: 3000 + parseInt(process.env.PIKO_ENV_ID || 0) }
```

```bash
# Or in shell
npm run dev -- --port $((3000 + PIKO_ENV_ID))
```

### SQLite / Local Databases

```bash
DATABASE_PATH=${PIKO_DATA_DIR}/dev.db
```

### Generated Binaries

```bash
go build -o ${PIKO_DATA_DIR}/bin/myapp
```

### Global Caches (if needed)

```bash
npm_config_cache=${PIKO_DATA_DIR}/.npm npm install
CARGO_HOME=${PIKO_DATA_DIR}/.cargo cargo build
```

## Implementation Plan

### Phase 1: Data Directory
- [ ] Add `PIKO_DATA_DIR` to env vars
- [ ] Create `.piko/data/<env>/` on `piko create`
- [ ] Clean up data directory on `piko destroy`

### Phase 2: Simple Mode
- [ ] Mode detection (no docker-compose.yml → simple mode)
- [ ] Skip Docker operations in simple mode
- [ ] Update `list`/`status` for simple mode

## Example: Piko Developing Itself

```yaml
# .piko.yml for piko project
scripts:
  prepare: "go mod download"
  run: |
    go run ./cmd/piko server --port $((19876 + PIKO_ENV_ID))

shells:
  dev: "go run ./cmd/piko server --port $((19876 + PIKO_ENV_ID))"
  test: "go test ./..."
```

Each worktree gets its own port, its own data dir. Zero config for isolation.

## Non-Goals

- Port management (users derive from PIKO_ENV_ID)
- Process management (use overmind/foreman if needed)
- Virtual environments (use direnv/asdf)
- Container orchestration (use Docker mode)

**Piko provides: worktrees + tmux + env vars + data dirs**

Users compose these primitives for their needs.
