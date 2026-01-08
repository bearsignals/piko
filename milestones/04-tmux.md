# Milestone 4: Tmux Integration

**Priority:** High
**Depends on:** M1 (Init & Create), M2 (Lifecycle)
**Unlocks:** Complete terminal workflow

## Goal

Each environment gets a tmux session with windows for shell, each service, and logs. Easy attach/switch between environments.

## Success Criteria

```bash
$ piko create feature-auth
✓ Created worktree
✓ Started containers
✓ Created tmux session piko/myapp/feature-auth
→ Attaching...

# Now inside tmux:
# Window 0: shell (at worktree path)
# Window 1: app (docker exec into app container)
# Window 2: db (docker exec into db with psql)
# Window 3: logs (docker compose logs -f)

$ piko switch feature-pay    # from inside tmux
# Switches to other session

$ piko attach feature-auth   # from outside tmux
# Attaches to session
```

## Tasks

### 3.1 Session Creation
- [ ] Create session: `tmux new-session -d -s "piko/<project>/<name>"`
- [ ] Set default directory to worktree path
- [ ] Store session name in SQLite

### 3.2 Window: Shell
- [ ] First window named "shell"
- [ ] Runs user's shell at worktree path
- [ ] Command: `cd <worktree> && $SHELL`

### 3.3 Window: Services
- [ ] Parse services from compose config
- [ ] Create window for each service
- [ ] Command: `docker compose -p <project> exec <service> <shell>`
- [ ] Shell lookup order:
  1. `shells.<service>` from `.piko.yml` (e.g., `db: psql -U postgres`)
  2. Default: `sh`
- [ ] No automatic inference from image names (explicit > magic)

### 3.4 Window: Logs
- [ ] Final window named "logs"
- [ ] Command: `docker compose -p <project> logs -f`
- [ ] Shows all service logs interleaved

### 3.5 Attach Command
- [ ] `piko attach <name>` — attach to session
- [ ] If already in tmux: error with suggestion to use `switch`
- [ ] If session doesn't exist: offer to run `piko up` first
- [ ] Command: `tmux attach -t "piko/<project>/<name>"`

### 3.6 Switch Command
- [ ] `piko switch <name>` — switch session (must be in tmux)
- [ ] If not in tmux: error with suggestion to use `attach`
- [ ] Command: `tmux switch-client -t "piko/<project>/<name>"`

### 3.7 Pick Command
- [ ] `piko pick` — fuzzy picker for sessions
- [ ] Requires fzf
- [ ] Lists all piko sessions
- [ ] Attaches/switches to selected

### 3.8 Session Cleanup
- [ ] On `piko down`: keep session (user might want shell)
- [ ] On `piko destroy`: kill session
- [ ] Command: `tmux kill-session -t "piko/<project>/<name>"`

### 3.9 Integrate with Create
- [ ] After containers start, create session
- [ ] After session created, attach (unless `--no-attach` flag)

## Test Cases

1. **Create with tmux**: Session created, attached
2. **Create --no-attach**: Session created, not attached
3. **Attach from outside**: Attaches successfully
4. **Attach from inside tmux**: Error with switch suggestion
5. **Switch from inside**: Switches successfully
6. **Switch from outside tmux**: Error with attach suggestion
7. **Pick with fzf**: Lists and selects
8. **Pick without fzf**: Error with install suggestion
9. **Destroy**: Session killed
10. **Windows correct**: Shell + services + logs

## Definition of Done

- [ ] `piko create` creates tmux session
- [ ] Session has shell window at worktree
- [ ] Session has window per service with correct shell
- [ ] Session has logs window
- [ ] `piko attach` works from outside tmux
- [ ] `piko switch` works from inside tmux
- [ ] `piko pick` works with fzf
- [ ] `piko destroy` kills session
