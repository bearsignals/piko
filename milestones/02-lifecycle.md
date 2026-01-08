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
- [ ] Query all environments from SQLite
- [ ] For each, check container status via `docker compose ps`
- [ ] Format as table with: name, status, branch, created
- [ ] Handle: no environments yet

### 2.2 Down Command
- [ ] Validate environment exists
- [ ] Run `docker compose -p <project> down` from worktree
- [ ] Keep worktree and database entry intact
- [ ] Report success/failure

### 2.3 Up Command
- [ ] Validate environment exists
- [ ] Run `docker compose -p <project> up -d` from worktree
- [ ] Regenerate override if needed (compose file might have changed)
- [ ] Report success/failure

### 2.4 Restart Command
- [ ] `piko restart <name>` — restart all services
- [ ] `piko restart <name> <service>` — restart specific service
- [ ] Use `docker compose restart`

### 2.5 Destroy Command
- [ ] Validate environment exists
- [ ] Run destroy script (if defined in .piko.yml):
  - Export PIKO_* environment variables
  - cd to worktree directory
  - Execute `scripts.destroy`
  - Continue even if script fails (warn user)
- [ ] Stop containers: `docker compose down`
- [ ] Optional `--volumes` flag: `docker compose down -v`
- [ ] Remove worktree: `git worktree remove <path>`
- [ ] Delete from SQLite
- [ ] Remove generated files

### 2.6 Status Helpers
- [ ] Function to check if containers are running
- [ ] Function to get container health status
- [ ] Cache/memoize for list command performance

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

- [ ] `piko list` shows all environments with status
- [ ] `piko down <name>` stops containers
- [ ] `piko up <name>` starts containers
- [ ] `piko restart <name>` restarts containers
- [ ] `piko destroy <name>` removes everything
- [ ] `piko destroy <name> --volumes` also removes data
- [ ] All commands are idempotent
