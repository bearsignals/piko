# piko — Worktree Development Environments

## Overview

`piko` is a CLI tool that creates isolated development environments for each git worktree, orchestrating Docker containers and tmux sessions to enable seamless parallel development.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   "I want to work on feature-auth and feature-payments simultaneously,     │
│    each with their own database state, without conflicts."                  │
│                                                                             │
│   $ piko create feature-auth                                                │
│   $ piko create feature-payments                                            │
│   $ piko attach feature-auth        ← now in isolated env with containers  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                USER LAYER                                   │
├──────────────────┬──────────────────┬──────────────────┬────────────────────┤
│                  │                  │                  │                    │
│   Terminal       │   Editor         │   Browser        │   Piko UI          │
│   (tmux)         │   (Cursor, etc)  │   (localhost)    │   (localhost:19876)│
│                  │                  │                  │                    │
└────────┬─────────┴────────┬─────────┴────────┬─────────┴──────────┬─────────┘
         │                  │                  │                    │
         │ attach/switch    │ file edits       │ http               │ manage
         │                  │                  │                    │
┌────────▼──────────────────▼──────────────────▼────────────────────▼─────────┐
│                              piko CLI + Server                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │   create    │ │    up       │ │   attach    │ │   server (web UI)   │   │
│  │   destroy   │ │   down      │ │   switch    │ │   list / status     │   │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────────┤
│                            CORE MANAGERS                                    │
│  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐            │
│  │  Git Manager     │ │  Docker Manager  │ │  Tmux Manager    │            │
│  │                  │ │                  │ │                  │            │
│  │  • worktree ops  │ │  • compose up    │ │  • session ops   │            │
│  │  • branch track  │ │  • port alloc    │ │  • window ops    │            │
│  │                  │ │  • network mgmt  │ │  • attach/switch │            │
│  └────────┬─────────┘ └────────┬─────────┘ └────────┬─────────┘            │
│           │                    │                    │                       │
├───────────▼────────────────────▼────────────────────▼───────────────────────┤
│                          INFRASTRUCTURE                                     │
│                                                                             │
│   ┌─────────────┐      ┌─────────────┐      ┌─────────────┐                │
│   │     Git     │      │   Docker    │      │    Tmux     │                │
│   │             │      │   Engine    │      │   Server    │                │
│   └─────────────┘      └─────────────┘      └─────────────┘                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Data Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Project                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │  name: "pouch"                                                        │ │
│  │  root: /home/user/pouch                                               │ │
│  │  compose_file: docker-compose.yml                                     │ │
│  │  config: .piko.yml (optional)                                         │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│       │                                                                     │
│       │ has many                                                            │
│       ▼                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                           Environment                                  │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐  │ │
│  │  │  name: "feature-auth"                                           │  │ │
│  │  │  branch: "feature/auth"                                         │  │ │
│  │  │  path: /home/user/pouch/.piko/worktrees/feature-auth            │  │ │
│  │  │  status: running | stopped                                      │  │ │
│  │  │                                                                 │  │ │
│  │  │  docker_project: "piko-pouch-feature-auth"                      │  │ │
│  │  │  tmux_session: "piko/pouch/feature-auth"                        │  │ │
│  │  │  ports: {app: 52341, db: 52342}  ← dynamically assigned         │  │ │
│  │  └─────────────────────────────────────────────────────────────────┘  │ │
│  │       │                                                                │ │
│  │       │ has many                                                       │ │
│  │       ▼                                                                │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐  │ │
│  │  │                          Service                                │  │ │
│  │  │  name: "app"                                                    │  │ │
│  │  │  image: "pouch-app" (built) | "postgres:15" (pulled)            │  │ │
│  │  │  container: "piko-pouch-feature-auth-app-1"                     │  │ │
│  │  │  shared: false                                                  │  │ │
│  │  │  shell: "sh" | "psql -U postgres" | "redis-cli"                 │  │ │
│  │  └─────────────────────────────────────────────────────────────────┘  │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│       │ has many (optional)                                                 │
│       ▼                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        Shared Service                                  │ │
│  │  ┌─────────────────────────────────────────────────────────────────┐  │ │
│  │  │  name: "redis"                                                  │  │ │
│  │  │  container: "piko-shared-redis-1"                               │  │ │
│  │  │  network: "piko-shared"                                         │  │ │
│  │  │  consumers: ["feature-auth", "feature-payments"]                │  │ │
│  │  └─────────────────────────────────────────────────────────────────┘  │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## File System Layout

```
project-root/
│
├── .git/                           # git directory
├── .piko.yml                       # optional piko config
├── docker-compose.yml              # existing compose file (required)
│
├── src/                            # your code (main branch)
├── ...
│
└── .piko/                          # piko directory
    │
    ├── state.db                    # SQLite database (state, history)
    │
    └── worktrees/
        │
        ├── feature-auth/               # worktree: feature/auth branch
        │   ├── .git                    # file pointing to ../../../.git
        │   ├── docker-compose.piko.yml # generated overrides (ports/networks only)
        │   ├── docker-compose.yml      # from git (same as main, used for volumes)
        │   ├── src/                    # code (feature/auth branch)
        │   └── ...
        │
        └── feature-payments/           # worktree: feature/payments branch
            ├── .git
            ├── docker-compose.piko.yml
            ├── docker-compose.yml
            ├── src/
            └── ...
```

---

## Docker Architecture

### Isolation Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            DOCKER ENGINE                                    │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────┐    │
│  │  Network: piko-shared                                               │    │
│  │                                                                     │    │
│  │                    ┌───────────────┐                                │    │
│  │                    │ redis         │  shared service                │    │
│  │                    │ :6379         │  (one instance)                │    │
│  │                    └───────┬───────┘                                │    │
│  │                            │                                        │    │
│  └────────────────────────────┼────────────────────────────────────────┘    │
│                               │                                             │
│              ┌────────────────┼────────────────┐                            │
│              │                │                │                            │
│  ┌───────────▼────────────┐  │  ┌─────────────▼──────────────┐             │
│  │  Network:              │  │  │  Network:                  │             │
│  │  piko-pouch-feature-auth│  │  │  piko-pouch-feature-payments│            │
│  │                        │  │  │                            │             │
│  │  ┌──────┐  ┌──────┐    │  │  │  ┌──────┐  ┌──────┐        │             │
│  │  │ app  │  │  db  │    │  │  │  │ app  │  │  db  │        │             │
│  │  │:52341│  │:52342│    │  │  │  │:52380│  │:52381│        │             │
│  │  └──┬───┘  └──────┘    │  │  │  └──┬───┘  └──────┘        │             │
│  │     │  (dynamic ports) │  │  │     │  (dynamic ports)     │             │
│  └─────┼──────────────────┘  │  └─────┼──────────────────────┘             │
│        │                     │        │                                     │
│        │ volume mount        │        │ volume mount                        │
│        │                     │        │                                     │
└────────┼─────────────────────┼────────┼─────────────────────────────────────┘
         │                     │        │
         ▼                     │        ▼
┌────────────────────────────┐ │  ┌────────────────────────────┐
│ .piko/worktrees/           │ │  │ .piko/worktrees/           │
│   feature-auth/src/        │ │  │   feature-payments/src/    │
└────────────────────────────┘ │  └────────────────────────────┘
                               │
                         (network link)
```

### Port Allocation Strategy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Deterministic Port Allocation                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Strategy: Calculate ports from worktree ID (from SQLite)                  │
│                                                                             │
│   Formula:                                                                  │
│     base_port = 10000 + (worktree_id * 100)                                 │
│     host_port = base_port + (container_port % 100)                          │
│                                                                             │
│   Example (worktree_id=1, base=10100):                                      │
│     db:5432      → 10132                                                    │
│     jaeger:4318  → 10118                                                    │
│     app:8080     → 10180                                                    │
│     riverui:8081 → 10181                                                    │
│                                                                             │
│   Example (worktree_id=2, base=10200):                                      │
│     db:5432      → 10232                                                    │
│     jaeger:4318  → 10218                                                    │
│     app:8080     → 10280                                                    │
│                                                                             │
│   Benefits:                                                                 │
│     • Deterministic: same worktree always gets same ports                   │
│     • No conflicts: worktree ID guarantees uniqueness                       │
│     • Predictable: user can memorize or script around them                  │
│     • Local dev friendly: ports work for host-based app development         │
│                                                                             │
│   Discovery via CLI:                                                        │
│     $ piko ports feature-auth                                               │
│     SERVICE   CONTAINER PORT   HOST PORT                                    │
│     db        5432             10132                                        │
│     jaeger    4318             10118                                        │
│     app       8080             10180                                        │
│                                                                             │
│     $ piko ports feature-auth --env                                         │
│     DB_PORT=10132                                                           │
│     JAEGER_PORT=10118                                                       │
│     APP_PORT=10180                                                          │
│                                                                             │
│     $ eval $(piko ports feature-auth --env)                                 │
│     $ make run   # app connects to localhost:$DB_PORT                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Compose Override Generation

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Zero-Config Volume Handling                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Key insight: Git worktree copies docker-compose.yml to the worktree.     │
│   Running compose FROM the worktree makes relative paths resolve correctly. │
│                                                                             │
│   Original (project root):                                                  │
│     /project/docker-compose.yml                                             │
│     volumes: ["./src:/app"]  → mounts /project/src                          │
│                                                                             │
│   Worktree (copied by git):                                                 │
│     /project/.piko/worktrees/feature-auth/docker-compose.yml               │
│     volumes: ["./src:/app"]  → mounts /project/.piko/worktrees/.../src     │
│                                                                             │
│   ✓ No user changes required                                                │
│   ✓ No path rewriting                                                       │
│   ✓ Original compose file works unchanged                                   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Override file only handles ports and networks:                            │
│                                                                             │
│   # docker-compose.piko.yml (generated, ~15 lines)                          │
│   services:                                                                 │
│     app:                                                                    │
│       ports:                                                                │
│         - "8080"              # removes host binding                        │
│     db:                                                                     │
│       ports:                                                                │
│         - "5432"                                                            │
│                                                                             │
│   networks:                                                                 │
│     default:                                                                │
│       name: piko-pouch-feature-auth                                         │
│     piko-shared:              # only if using shared services               │
│       external: true                                                        │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Generation uses Docker's own parser:                                      │
│                                                                             │
│   $ docker compose config --format json                                     │
│     → Resolves extends, profiles, env vars, multiple files                  │
│     → Outputs normalized JSON                                               │
│     → Piko parses this, extracts ports, generates override                  │
│                                                                             │
│   No custom YAML parsing. Docker handles complexity.                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Scripts (Lifecycle Hooks)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SCRIPTS PRIMITIVE                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Scripts let projects define what happens at each lifecycle stage.        │
│   Piko exposes environment variables; scripts use them.                    │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  SETUP SCRIPT                                                       │  │
│   │  Runs after: worktree created, containers started                   │  │
│   │  Use for: symlink env files, install deps, run migrations           │  │
│   │                                                                     │  │
│   │  scripts:                                                           │  │
│   │    setup: |                                                         │  │
│   │      ln -s "$PIKO_ROOT/.env.local" .env.local                       │  │
│   │      go mod download                                                │  │
│   │      make migrate                                                   │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  RUN SCRIPT                                                         │  │
│   │  Triggered by: piko run <name>                                      │  │
│   │  Use for: start dev server with correct env vars                    │  │
│   │                                                                     │  │
│   │  scripts:                                                           │  │
│   │    run: |                                                           │  │
│   │      DATABASE_URL="postgres://user:pass@localhost:$PIKO_DB_PORT/db" │  │
│   │      go run cmd/main.go                                             │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  DESTROY SCRIPT                                                     │  │
│   │  Runs before: worktree removal                                      │  │
│   │  Use for: cleanup external resources                                │  │
│   │                                                                     │  │
│   │  scripts:                                                           │  │
│   │    destroy: |                                                       │  │
│   │      echo "Cleaning up $PIKO_ENV_NAME"                              │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Why scripts instead of magic?                                             │
│                                                                             │
│   Before: Piko detects env_file in compose, auto-symlinks, infers shells   │
│   After:  Piko exposes $PIKO_* vars, project declares what it needs        │
│                                                                             │
│   • Explicit > implicit                                                     │
│   • Works with any project structure                                        │
│   • No surprises from inference                                             │
│   • Composable with existing tools (make, npm, etc)                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Environment Variables

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PIKO ENVIRONMENT VARIABLES                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Available in all scripts:                                                 │
│                                                                             │
│   Variable              │  Example Value                                    │
│   ──────────────────────┼─────────────────────────────────────────────────  │
│   PIKO_ROOT             │  /home/user/pouch                                 │
│   PIKO_ENV_NAME         │  feature-auth                                     │
│   PIKO_ENV_PATH         │  /home/user/pouch/.piko/worktrees/feature-auth    │
│   PIKO_PROJECT          │  piko-pouch-feature-auth                          │
│   PIKO_BRANCH           │  feature/auth                                     │
│                                                                             │
│   Dynamic port variables (one per service with exposed ports):              │
│                                                                             │
│   PIKO_<SERVICE>_PORT   │  PIKO_DB_PORT=10132                               │
│                         │  PIKO_APP_PORT=10180                              │
│                         │  PIKO_JAEGER_PORT=10118                           │
│                         │  PIKO_RIVERUI_PORT=10181                          │
│                                                                             │
│   Service names are uppercased, hyphens become underscores:                 │
│     my-service → PIKO_MY_SERVICE_PORT                                       │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Usage in scripts:                                                         │
│                                                                             │
│   scripts:                                                                  │
│     run: |                                                                  │
│       DATABASE_URL="postgres://user:pass@localhost:$PIKO_DB_PORT/mydb" \    │
│       OTEL_ENDPOINT="http://localhost:$PIKO_JAEGER_PORT" \                  │
│       go run cmd/main.go                                                    │
│                                                                             │
│   Usage from shell:                                                         │
│                                                                             │
│   $ piko env feature-auth                  # print all vars                 │
│   $ eval $(piko env feature-auth)          # export to current shell        │
│   $ piko env feature-auth --json           # JSON format                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Tmux Architecture

### Session Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TMUX SERVER                                       │
│                         (default server)                                    │
│                                                                             │
│   User's existing sessions (untouched):                                     │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  main                                                                │  │
│   │  work                                                                │  │
│   │  ...                                                                 │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   Worktree sessions (prefixed with piko/<project>/):                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                                                                      │  │
│   │  Session: piko/pouch/feature-auth                                    │  │
│   │  ├── Window 0: shell     → local shell at worktree path              │  │
│   │  ├── Window 1: app       → docker exec ... app sh                    │  │
│   │  ├── Window 2: db        → docker exec ... db psql -U postgres       │  │
│   │  └── Window 3: logs      → docker compose logs -f                    │  │
│   │                                                                      │  │
│   │  Session: piko/pouch/feature-payments                                │  │
│   │  ├── Window 0: shell                                                 │  │
│   │  ├── Window 1: app                                                   │  │
│   │  ├── Window 2: db                                                    │  │
│   │  └── Window 3: logs                                                  │  │
│   │                                                                      │  │
│   │  Session: piko/acme/feature-x    ← different project                 │  │
│   │  ├── Window 0: shell                                                 │  │
│   │  ├── Window 1: api                                                   │  │
│   │  └── Window 2: logs                                                  │  │
│   │                                                                      │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   Ctrl-b s (session picker) shows:                                          │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  (0)   main                                                          │  │
│   │  (1)   work                                                          │  │
│   │  (2) + piko/acme/feature-x: 3 windows                                │  │
│   │  (3) + piko/pouch/feature-auth: 4 windows (attached)                 │  │
│   │  (4) + piko/pouch/feature-payments: 4 windows                        │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Window Commands

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Window Type     │  Command                                                 │
├──────────────────┼──────────────────────────────────────────────────────────┤
│  shell           │  cd /path/to/worktree && $SHELL                          │
│  app (service)   │  docker compose -p PROJECT exec app sh                   │
│  db (service)    │  docker compose -p PROJECT exec db psql -U postgres      │
│  logs            │  docker compose -p PROJECT logs -f                       │
│  custom          │  user-defined command                                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Code ↔ Container Sync

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│                            EDIT → RELOAD FLOW                               │
│                                                                             │
│   ┌─────────────┐         ┌─────────────┐         ┌─────────────┐          │
│   │   Editor    │         │    Host     │         │  Container  │          │
│   │  (VS Code)  │         │ Filesystem  │         │    (app)    │          │
│   └──────┬──────┘         └──────┬──────┘         └──────┬──────┘          │
│          │                       │                       │                  │
│          │  save file            │                       │                  │
│          │──────────────────────▶│                       │                  │
│          │                       │                       │                  │
│          │                       │  volume mount         │                  │
│          │                       │  (instant sync)       │                  │
│          │                       │──────────────────────▶│                  │
│          │                       │                       │                  │
│          │                       │                       │  detect change   │
│          │                       │                       │  (fs watcher)    │
│          │                       │                       │                  │
│          │                       │                       │  hot reload      │
│          │                       │                       │  (dev server)    │
│          │                       │                       │                  │
│          │                       │                 200ms │                  │
│          │◀──────────────────────────────────────────────│                  │
│          │              refresh / HMR                    │                  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

Requirements:
  1. Volume mount in docker-compose.yml:  volumes: [".:/app"]
  2. Dev server with hot reload:          command: npm run dev
```

---

## Configuration

### Layer 1: Inferred (Zero Config)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AUTOMATICALLY INFERRED                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Source                      │  Inferred Value                             │
│   ────────────────────────────┼─────────────────────────────────────────    │
│   docker-compose.yml          │  services, ports, networks                  │
│   directory name              │  project name                               │
│   git branch                  │  environment name                           │
│   service count               │  tmux windows                               │
│   compose ports               │  PIKO_<SERVICE>_PORT variables              │
│                                                                             │
│   NOT inferred (configure explicitly):                                      │
│   ────────────────────────────┼─────────────────────────────────────────    │
│   container shells            │  configure in shells: {db: psql -U x}       │
│   env file locations          │  symlink in scripts.setup                   │
│   migrations/setup            │  run in scripts.setup                       │
│   dev server                  │  define in scripts.run                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Layer 2: Project Config (Optional)

```yaml
# .piko.yml — only create if you need to customize

# Lifecycle scripts (the primary configuration mechanism)
scripts:
  setup: |
    ln -s "$PIKO_ROOT/.env.local" .env.local
    go mod download
    make migrate

  run: |
    DATABASE_URL="postgres://user:password@localhost:$PIKO_DB_PORT/pouch?sslmode=disable" \
    OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:$PIKO_JAEGER_PORT" \
    go run cmd/accounts/main.go

  destroy: |
    echo "Cleaned up $PIKO_ENV_NAME"

# Services shared across all worktrees (not isolated)
shared:
  - jaeger

# Custom shell commands for tmux windows (exec into containers)
shells:
  db: psql -U user -d pouch
  redis: redis-cli

# Services to exclude from tmux windows
ignore:
  - riverui

# Additional tmux windows
windows:
  - name: frontend
    command: cd frontend && npm run dev
    local: true  # not a container
```

### Layer 3: User Config (Optional)

```yaml
# ~/.config/piko/config.yml — personal preferences

# Tmux session prefix (default: piko/)
session_prefix: "dev:"

# Default shell when not inferred
default_shell: bash

# Editor command
editor: cursor

# Always include these windows
default_windows:
  - name: shell
    local: true
  - name: logs
    command: docker compose -p ${PROJECT} logs -f
```

### Configuration Merge Order

```
┌─────────────────┐
│  User Config    │  ~/.config/piko/config.yml
└────────┬────────┘
         │ overridden by
         ▼
┌─────────────────┐
│ Project Config  │  .piko.yml
└────────┬────────┘
         │ overridden by
         ▼
┌─────────────────┐
│   CLI Flags     │  piko create --shared redis
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Final Config   │
└─────────────────┘
```

---

## CLI Commands

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              piko CLI                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  LIFECYCLE                                                                  │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko create <name> [--branch <branch>]   Create worktree + env            │
│  piko destroy <name> [--volumes]          Remove worktree + env + data     │
│                                                                             │
│  ENVIRONMENT CONTROL                                                        │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko up <name>                           Start containers + tmux session  │
│  piko down <name>                         Stop containers (keep session)   │
│  piko restart <name> [service]            Restart containers               │
│                                                                             │
│  CONTEXT SWITCHING                                                          │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko attach <name>                       Attach to tmux session           │
│  piko switch <name>                       Switch tmux session (if in tmux) │
│  piko pick                                Interactive fuzzy picker         │
│                                                                             │
│  SCRIPTS                                                                    │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko run <name>                          Execute run script               │
│  piko env <name>                          Print environment variables      │
│  piko env <name> --json                   Print env vars as JSON           │
│                                                                             │
│  INSPECTION                                                                 │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko list                                List all environments            │
│  piko status [name]                       Detailed status                  │
│  piko logs <name> [service] [-f]          View logs                        │
│  piko open <name> [service]               Open in browser (discovers port) │
│                                                                             │
│  INTERACTION                                                                │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko exec <name> <service> [cmd]         Exec into container              │
│  piko shell <name> <service>              Interactive shell in container   │
│                                                                             │
│  EDITOR                                                                     │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko edit <name>                         Open worktree in editor (Cursor) │
│  piko edit --all                          Open workspace with all worktrees│
│                                                                             │
│  SHARED SERVICES                                                            │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko share <service>                     Make service shared              │
│  piko isolate <service>                   Make service isolated            │
│                                                                             │
│  SERVER                                                                     │
│  ─────────────────────────────────────────────────────────────────────────  │
│  piko server                              Start web UI server (:19876)     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Command Flow: `piko create feature-auth`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   $ piko create feature-auth                                                │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 1. VALIDATE                                                         │  │
│   │    • Check docker-compose.yml exists                                │  │
│   │    • Check git repo                                                 │  │
│   │    • Check name not already used                                    │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 2. GIT WORKTREE                                                     │  │
│   │    • git worktree add .piko/worktrees/feature-auth -b feature-auth  │  │
│   │    • Or use existing branch if specified                            │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 3. GENERATE COMPOSE OVERRIDE                                        │  │
│   │    • Run: docker compose config --format json                       │  │
│   │    • Parse services and ports from normalized output                │  │
│   │    • Generate docker-compose.piko.yml (ports + networks only)       │  │
│   │    • Link to shared network if needed                               │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 4. START CONTAINERS                                                 │  │
│   │    • cd .piko/worktrees/feature-auth                                │  │
│   │    • docker compose -p piko-pouch-feature-auth \                    │  │
│   │        -f docker-compose.yml \                                      │  │
│   │        -f docker-compose.piko.yml \                                 │  │
│   │        up -d                                                        │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 5. CREATE TMUX SESSION                                              │  │
│   │    • tmux new-session -d -s "piko/pouch/feature-auth" -n shell      │  │
│   │    • Add window for each service (with exec command)                │  │
│   │    • Add logs window                                                │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 6. RUN SETUP SCRIPT                                                 │  │
│   │    • Export PIKO_* environment variables                            │  │
│   │    • cd to worktree directory                                       │  │
│   │    • Execute scripts.setup from .piko.yml (if defined)              │  │
│   │    • Example: symlink .env.local, run migrations                    │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 7. UPDATE STATE                                                     │  │
│   │    • Insert into .piko/state.db (SQLite)                            │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 8. ATTACH                                                           │  │
│   │    • tmux attach -t "piko/pouch/feature-auth"                       │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   Output:                                                                   │
│   ✓ Created worktree at .piko/worktrees/feature-auth (branch: feature-auth)│
│   ✓ Started 3 containers (app, db, jaeger:shared)                          │
│   ✓ Ran setup script                                                       │
│   ✓ Created tmux session piko/pouch/feature-auth                           │
│   → Attached to session                                                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Command Flow: `piko run feature-auth`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   $ piko run feature-auth                                                   │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 1. RESOLVE ENVIRONMENT                                              │  │
│   │    • Look up feature-auth in .piko/state.db                         │  │
│   │    • Verify containers are running (start if not)                   │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 2. DISCOVER PORTS                                                   │  │
│   │    • docker compose -p piko-pouch-feature-auth port db 5432         │  │
│   │    • docker compose -p piko-pouch-feature-auth port jaeger 4318     │  │
│   │    • ... for each service with exposed ports                        │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 3. BUILD ENVIRONMENT                                                │  │
│   │    • PIKO_ROOT=/home/user/pouch                                     │  │
│   │    • PIKO_ENV_NAME=feature-auth                                     │  │
│   │    • PIKO_ENV_PATH=/home/user/pouch/.piko/worktrees/feature-auth    │  │
│   │    • PIKO_PROJECT=piko-pouch-feature-auth                           │  │
│   │    • PIKO_DB_PORT=10132                                             │  │
│   │    • PIKO_JAEGER_PORT=10118                                         │  │
│   │    • ...                                                            │  │
│   └──────────────────────────────────┬──────────────────────────────────┘  │
│                                      ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │ 4. EXECUTE RUN SCRIPT                                               │  │
│   │    • cd to worktree directory                                       │  │
│   │    • Export PIKO_* variables                                        │  │
│   │    • Execute scripts.run from .piko.yml                             │  │
│   │    • Stream output to terminal                                      │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   Example .piko.yml:                                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  scripts:                                                           │  │
│   │    run: |                                                           │  │
│   │      DATABASE_URL="postgres://u:p@localhost:$PIKO_DB_PORT/db" \     │  │
│   │      go run cmd/main.go                                             │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## State Management

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  .piko/state.db — SQLite database                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Schema:                                                                    │
│                                                                             │
│  CREATE TABLE project (                                                     │
│      id INTEGER PRIMARY KEY,                                                │
│      name TEXT UNIQUE NOT NULL,                                             │
│      root_path TEXT NOT NULL,                                               │
│      compose_file TEXT DEFAULT 'docker-compose.yml',                        │
│      created_at DATETIME DEFAULT CURRENT_TIMESTAMP                          │
│  );                                                                         │
│                                                                             │
│  CREATE TABLE environments (                                                │
│      id INTEGER PRIMARY KEY,                                                │
│      project_id INTEGER REFERENCES project(id) ON DELETE CASCADE,           │
│      name TEXT NOT NULL,                                                    │
│      branch TEXT NOT NULL,                                                  │
│      path TEXT NOT NULL,                                                    │
│      docker_project TEXT NOT NULL,                                          │
│      tmux_session TEXT NOT NULL,                                            │
│      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,                         │
│      UNIQUE(project_id, name)                                               │
│  );                                                                         │
│                                                                             │
│  CREATE TABLE shared_services (                                             │
│      id INTEGER PRIMARY KEY,                                                │
│      project_id INTEGER REFERENCES project(id) ON DELETE CASCADE,           │
│      service_name TEXT NOT NULL,                                            │
│      container_name TEXT,                                                   │
│      network TEXT NOT NULL,                                                 │
│      UNIQUE(project_id, service_name)                                       │
│  );                                                                         │
│                                                                             │
│  Benefits over JSON:                                                        │
│    • ACID transactions — safe concurrent access                             │
│    • No file corruption from partial writes                                 │
│    • Query capabilities for complex operations                              │
│    • Single file, no external database server                               │
│                                                                             │
│  Note: Port mappings are not stored — discovered at runtime via             │
│        `docker compose port` since they're dynamically assigned.            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Worktree location | `.piko/worktrees/<name>` | Keeps everything in project, easy to gitignore |
| Docker isolation | Project name prefix | Native compose feature, no hacks |
| Port allocation | Deterministic (ID-based) | Predictable, scriptable, local dev friendly |
| Tmux sessions | One per worktree | Natural mental model, easy switching |
| Tmux server | Default (shared) | Works with existing user setup |
| Session naming | `piko/<project>/<name>` | Namespaced by project, visible in `tmux ls` |
| Config format | YAML | Familiar, matches compose |
| State storage | SQLite | ACID safe, handles concurrent access |
| Shared services | Separate network | Clean isolation, explicit linking |
| Volume mounts | Zero-config (git copies compose) | No user changes, paths resolve naturally |
| Compose parsing | `docker compose config` | Offloads complexity to Docker itself |
| Lifecycle hooks | Scripts primitive | Explicit > magic; project declares needs |
| Env file handling | Via setup script | No auto-detection; user symlinks explicitly |
| Port injection | `$PIKO_<SERVICE>_PORT` vars | Composable with any tooling (make, npm, etc) |
| Tmux shells | Explicit `shells` config | No inference from image; user declares commands |

---

## What's NOT Configurable (By Design)

These are fixed to keep the tool simple and predictable:

- Worktree directory structure (`.piko/worktrees/<name>/`)
- Docker project naming (`piko-<project>-<name>`)
- Tmux session naming (`piko/<project>/<name>`)
- State database location (`.piko/state.db`)
- Compose override filename (`docker-compose.piko.yml`)
- Shared network name (`piko-<project>-shared`)
- Web UI port (`19876`)

---

## Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            REQUIRED                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  • git        (any recent version with worktree support)                   │
│  • docker     (with compose v2)                                            │
│  • tmux       (any recent version)                                         │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                            OPTIONAL                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  • fzf        (for `piko pick` fuzzy finder)                               │
│  • cursor     (for `piko edit` editor integration)                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Web UI

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              PIKO WEB UI                                    │
│                           localhost:19876                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │  pouch                                                      [Refresh] │ │
│  ├───────────────────────────────────────────────────────────────────────┤ │
│  │                                                                       │ │
│  │  ● feature-auth                                          running     │ │
│  │    ├─ app    localhost:52341                                         │ │
│  │    ├─ db     localhost:52342                                         │ │
│  │    └─ redis  localhost:6379 (shared)                                 │ │
│  │                                                                       │ │
│  │    [Open in Cursor]  [View Ports]  [Logs]  [Stop]                    │ │
│  │                                                                       │ │
│  ├───────────────────────────────────────────────────────────────────────┤ │
│  │                                                                       │ │
│  │  ○ feature-payments                                      stopped     │ │
│  │                                                                       │ │
│  │    [Open in Cursor]  [Start]                                         │ │
│  │                                                                       │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Server: piko server                                                        │
│                                                                             │
│  API Endpoints:                                                             │
│    GET  /api/environments              List all environments               │
│    GET  /api/environments/:name        Get environment details + ports     │
│    POST /api/environments/:name/up     Start environment                   │
│    POST /api/environments/:name/down   Stop environment                    │
│    POST /api/environments/:name/open   Open in Cursor                      │
│                                                                             │
│  Implementation:                                                            │
│    • Single-file HTML/JS UI (no build step)                                │
│    • Embedded in piko binary                                               │
│    • On-demand: user runs `piko server` when needed                        │
│    • "Open in Cursor" calls: cursor <worktree-path>                        │
│    • Auto-refresh via polling or SSE                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```
