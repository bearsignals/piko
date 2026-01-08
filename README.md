# piko

Isolated dev environments from git worktrees.

```
piko create feature-auth      # worktree + containers + tmux
piko create feature-payments  # another isolated environment
piko attach feature-auth      # switch instantly
```

Each environment gets its own branch, containers, ports, and tmux session. No conflicts.

## Install

```
go install github.com/gwuah/piko@latest
```

Requires: git, tmux. Docker optional.

## Usage

```bash
# Initialize in your project
cd your-project
piko init

# Create environments
piko create my-feature           # new branch + environment
piko create --branch main prod   # existing branch

# Work
piko attach my-feature           # attach to tmux session
piko list                        # see all environments
piko status my-feature           # detailed status

# Lifecycle
piko up my-feature               # start containers
piko down my-feature             # stop containers
piko destroy my-feature          # remove everything
```

## How it works

**Docker mode** (project has docker-compose.yml):
- Creates git worktree at `.piko/worktrees/<name>/`
- Starts containers with unique project name and ports
- Creates tmux session with shell + service windows

**Simple mode** (no docker-compose.yml):
- Creates git worktree + data directory
- Provides `PIKO_ENV_ID` and `PIKO_DATA_DIR` for isolation
- You derive ports: `--port $((3000 + PIKO_ENV_ID))`

## Configuration

Optional `.piko.yml` in project root:

```yaml
scripts:
  setup: |
    ln -s "$PIKO_ROOT/.env.local" .env.local
    npm install
  run: |
    DATABASE_URL="postgres://localhost:$PIKO_DB_PORT/dev" npm run dev

shells:
  db: psql -U postgres
  redis: redis-cli
```

## Environment variables

Available in scripts and tmux sessions:

```
PIKO_ENV_NAME    # environment name
PIKO_ENV_ID      # unique integer (for port derivation)
PIKO_ENV_PATH    # worktree path
PIKO_DATA_DIR    # isolated data directory
PIKO_ROOT        # project root
PIKO_DB_PORT     # allocated port (docker mode)
```

## Web UI

```
piko server
```

Opens dashboard at `localhost:19876` to view/manage all environments.

## License

MIT
