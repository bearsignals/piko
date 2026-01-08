# Piko Milestones Overview

## Priority Order

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   M1: Init & Create ────► M2: Lifecycle ────► M3: Basic Server ────►       │
│       (foundation)         (up/down/destroy)   (browser create/open)        │
│                                                                             │
│   ────► M4: Tmux ────► M5: Inspection ────► M6: Full Web UI                │
│          (sessions)     (piko run, logs)      (ports, logs, controls)       │
│                                                                             │
│   ────► M7: Shared Services ────► M8: Configuration                         │
│          (redis, etc)              (shells, windows, user config)           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Milestone Summary

| # | Milestone | Priority | Effort | Delivers |
|---|-----------|----------|--------|----------|
| 1 | Init & Create | Critical | Medium | `piko init`, `piko create`, `piko env`, setup script |
| 2 | Lifecycle | Critical | Small | `up`, `down`, `destroy`, `list`, destroy script |
| 3 | Basic Server | High | Small | HTTP server, list, create from browser, open in Cursor |
| 4 | Tmux Integration | High | Medium | Sessions, attach, switch |
| 5 | Inspection | High | Small | `piko run`, `logs`, `status`, `open` |
| 6 | Full Web UI | Medium | Medium | Ports display, start/stop, logs in browser |
| 7 | Shared Services | Low | Medium | Cross-environment service sharing |
| 8 | Configuration | Low | Small | shells, windows, ignore, user config |

## Success Definition

**MVP (M1-M5 complete):**
```bash
# Initialize project
$ cd ~/projects/myapp
$ piko init
✓ Initialized piko in /home/user/myapp/.piko

# CLI workflow
$ piko create feature-auth
$ piko run feature-auth       # dev server with PIKO_* vars

# Browser workflow (localhost:19876)
$ piko server
# → Create new environments
# → Open in Cursor
```

**Usable Product (M1-M6 complete):**
- All core workflows functional
- Full web UI with ports, logs, start/stop
- Can manage multiple environments from browser

**Complete Product (M1-M8 complete):**
- Shared services for efficiency
- Customizable per-project and per-user
- Full scripts lifecycle (setup, run, destroy)

## Implementation Language

Go — single binary, good CLI libraries (cobra), embeds static files easily, SQLite support via CGO-free libraries (modernc.org/sqlite).

## Files

- `milestones/01-init-create.md`
- `milestones/02-lifecycle.md`
- `milestones/03-basic-server.md`
- `milestones/04-tmux.md`
- `milestones/05-inspection.md`
- `milestones/06-full-web-ui.md`
- `milestones/07-shared-services.md`
- `milestones/08-configuration.md`
