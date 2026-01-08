# Milestone 5: Inspection Commands

**Priority:** High
**Depends on:** M1 (Init & Create), M2 (Lifecycle)
**Unlocks:** `piko run` workflow, better visibility

## Goal

Commands to inspect environment details: ports, logs, status, and quick open in browser/editor.

## Success Criteria

```bash
$ piko env feature-auth
PIKO_ROOT=/home/user/myapp
PIKO_ENV_NAME=feature-auth
PIKO_ENV_PATH=/home/user/myapp/.piko/worktrees/feature-auth
PIKO_PROJECT=piko-myapp-feature-auth
PIKO_BRANCH=feature-auth
PIKO_APP_PORT=52341
PIKO_DB_PORT=52342
PIKO_REDIS_PORT=52343

$ piko run feature-auth
→ Running scripts.run from .piko.yml...
Server listening on :8080

$ piko status feature-auth
Environment: feature-auth
Branch:      feature-auth
Path:        .piko/worktrees/feature-auth
Docker:      piko-myapp-feature-auth
Tmux:        piko/myapp/feature-auth
Created:     2 hours ago
Status:      running (3/3 containers healthy)

CONTAINER                          STATUS    PORTS
piko-myapp-feature-auth-app-1      running   52341→8080
piko-myapp-feature-auth-db-1       running   52342→5432
piko-myapp-feature-auth-redis-1    running   52343→6379

$ piko open feature-auth app
→ Opening http://localhost:52341 in browser...

$ piko logs feature-auth -f
# Streams all logs

$ piko logs feature-auth app
# Shows only app logs

$ piko edit feature-auth
→ Opening .piko/worktrees/feature-auth in Cursor...
```

## Tasks

### 4.1 Run Command
- [x] `piko run <name>` — execute the run script
- [x] Validate environment exists and containers are running
- [x] If containers stopped, offer to start them first
- [x] Load `.piko.yml` from project root
- [x] If `scripts.run` is not defined: error with helpful message
- [x] Export PIKO_* environment variables
- [x] cd to worktree directory
- [x] Execute script via shell, stream output to terminal
- [x] Pass through signals (Ctrl+C)

### 4.2 Status Command
- [x] Show environment metadata from database
- [x] Show container status from Docker
- [x] Show health check status if available
- [x] Calculate "running X/Y containers"
- [x] Show port mappings inline

### 4.3 Open Command
- [x] `piko open <name>` — open first HTTP service
- [x] `piko open <name> <service>` — open specific service
- [x] Discover port via `docker compose port`
- [x] Open URL with system browser:
  - macOS: `open`
  - Linux: `xdg-open`
- [x] Error if service has no exposed ports

### 4.4 Logs Command
- [x] `piko logs <name>` — all services
- [x] `piko logs <name> <service>` — specific service
- [x] `-f` flag for follow mode
- [x] `--tail N` flag for last N lines
- [x] Proxy to `docker compose logs`

### 4.5 Edit Command
- [x] `piko edit <name>` — open worktree in editor
- [x] Detect editor: `$EDITOR`, then `cursor`, then `code`
- [x] Run: `cursor <worktree-path>`
- [ ] `piko edit --all` — open workspace with all worktrees (future)

### 4.6 Exec Command
- [x] `piko exec <name> <service> [cmd]`
- [x] Default cmd: detected shell for service
- [x] Proxy to `docker compose exec`
- [x] Interactive by default

### 4.7 Shell Command
- [x] `piko shell <name> <service>`
- [x] Shortcut for `piko exec <name> <service>` with detected shell
- [x] Always interactive

## Port Discovery

```go
func getPortMappings(project string) ([]PortMapping, error) {
    // docker compose -p <project> ps --format json
    // Parse: [{"Service":"app","Publishers":[{"TargetPort":8080,"PublishedPort":52341}]}]
}
```

## Test Cases

1. **Run with script**: Executes and streams output
2. **Run without script**: Error with helpful message
3. **Run containers stopped**: Offers to start
4. **Run Ctrl+C**: Properly terminates
5. **Status running**: Shows healthy status
6. **Status partial**: Shows "2/3 running"
7. **Open http service**: Opens browser
8. **Open non-http service**: Opens anyway (user's choice)
9. **Open stopped**: Error with suggestion
10. **Logs follow**: Streams continuously
11. **Logs specific service**: Filters correctly
12. **Edit**: Opens Cursor

## Definition of Done

- [x] `piko run <name>` executes run script with PIKO_* vars
- [x] `piko status <name>` shows detailed status
- [x] `piko open <name>` opens browser
- [x] `piko logs <name>` shows logs
- [x] `piko logs <name> -f` follows logs
- [x] `piko edit <name>` opens editor
- [x] `piko exec <name> <service>` runs command
- [x] `piko shell <name> <service>` opens shell
