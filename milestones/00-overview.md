# Piko Milestones Overview

## Priority Order

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   M1: Core Create Flow ────► M2: Lifecycle ────► M3: Tmux ────►            │
│         (foundation)          (up/down/destroy)   (sessions)               │
│                                                                             │
│   ────► M4: Inspection ────► M5: Web UI ────► M6: Shared Services          │
│          (ports/logs)         (browser UI)      (redis, etc)               │
│                                                                             │
│   ────► M7: Configuration                                                   │
│          (optional customization)                                           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Milestone Summary

| # | Milestone | Priority | Effort | Delivers |
|---|-----------|----------|--------|----------|
| 1 | Core Create Flow | Critical | Medium | `piko create`, `piko env`, setup script |
| 2 | Environment Lifecycle | Critical | Small | `up`, `down`, `destroy`, `list` |
| 3 | Tmux Integration | High | Medium | Sessions, attach, switch |
| 4 | Inspection Commands | Medium | Small | `piko run`, `logs`, `status`, `open` |
| 5 | Web UI | Medium | Medium | Browser-based management |
| 6 | Shared Services | Low | Medium | Cross-environment service sharing |
| 7 | Configuration | Low | Small | `.piko.yml`, scripts, user config |

## Success Definition

**MVP (M1-M3 complete):**
```bash
$ piko create feature-auth    # creates worktree, starts containers, runs setup script
$ piko attach feature-auth    # user is in tmux session
$ piko run feature-auth       # runs dev server with correct env vars
```

**Usable Product (M1-M5 complete):**
- All core workflows functional
- `piko env` for port discovery
- `piko run` for standardized dev workflow
- Can manage multiple environments comfortably

**Complete Product (M1-M7 complete):**
- Shared services for efficiency
- Customizable per-project and per-user
- Full scripts lifecycle (setup, run, destroy)

## Implementation Language

Go — single binary, good CLI libraries (cobra), embeds static files easily, SQLite support via CGO-free libraries (modernc.org/sqlite).

## Files

- `milestones/01-core-create.md`
- `milestones/02-lifecycle.md`
- `milestones/03-tmux.md`
- `milestones/04-inspection.md`
- `milestones/05-web-ui.md`
- `milestones/06-shared-services.md`
- `milestones/07-configuration.md`
