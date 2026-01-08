# Milestone 2: Environment Lifecycle

**Priority:** Critical
**Depends on:** M1 (Core Create)
**Unlocks:** Full environment management

## Goal

Complete lifecycle management: stop, start, restart, destroy environments. List all environments with their status.

## Success Criteria

```bash
$ piko list
NAME            STATUS     BRANCH          CREATED
feature-auth    running    feature-auth    2 hours ago
feature-pay     stopped    feature-pay     1 day ago

$ piko down feature-auth
✓ Stopped containers

$ piko up feature-auth
✓ Started containers

$ piko destroy feature-auth --volumes
✓ Stopped containers
✓ Removed volumes
✓ Removed worktree
✓ Removed from database
```

## Tasks

### 2.1 List Command
- [x] Query all environments from SQLite
- [x] For each, check container status via `docker compose ps`
- [x] Format as table with: name, status, branch, created
- [x] Handle: no environments yet

### 2.2 Down Command
- [x] Validate environment exists
- [x] Run `docker compose -p <project> down` from worktree
- [x] Keep worktree and database entry intact
- [x] Report success/failure

### 2.3 Up Command
- [x] Validate environment exists
- [x] Run `docker compose -p <project> up -d` from worktree
- [x] Regenerate override if needed (compose file might have changed)
- [x] Report success/failure

### 2.4 Restart Command
- [x] `piko restart <name>` — restart all services
- [x] `piko restart <name> <service>` — restart specific service
- [x] Use `docker compose restart`

### 2.5 Destroy Command
- [x] Validate environment exists
- [x] Run destroy script (if defined in .piko.yml):
  - Export PIKO_* environment variables
  - cd to worktree directory
  - Execute `scripts.destroy`
  - Continue even if script fails (warn user)
- [x] Stop containers: `docker compose down`
- [x] Optional `--volumes` flag: `docker compose down -v`
- [x] Remove worktree: `git worktree remove <path>`
- [x] Delete from SQLite
- [x] Remove generated files

### 2.6 Status Helpers
- [x] Function to check if containers are running
- [x] Function to get container health status
- [x] Cache/memoize for list command performance

## Test Cases

1. **List empty**: Shows helpful message
2. **List mixed**: Shows running and stopped correctly
3. **Down running**: Stops containers
4. **Down already stopped**: No error, idempotent
5. **Up stopped**: Starts containers
6. **Up already running**: No error, idempotent
7. **Destroy running**: Stops then destroys
8. **Destroy with volumes**: Removes Docker volumes
9. **Destroy non-existent**: Error with suggestion

## Definition of Done

- [x] `piko list` shows all environments with status
- [x] `piko down <name>` stops containers
- [x] `piko up <name>` starts containers
- [x] `piko restart <name>` restarts containers
- [x] `piko destroy <name>` removes everything
- [x] `piko destroy <name> --volumes` also removes data
- [x] All commands are idempotent
