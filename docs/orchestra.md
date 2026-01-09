# Orchestra: Real-Time Communication for Piko

## Overview

Orchestra enables real-time communication between the Piko server, web UI, and Claude Code instances running in tmux sessions. The primary use case is surfacing Claude Code input requests to users when working across multiple environments.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Claude Code (in tmux)                        │
│  > Permission needed: Edit file.go? [y/n]                       │
└───────────────────────────┬─────────────────────────────────────┘
                            │ Notification hook (non-blocking)
                            ▼
┌───────────────────────────────────────────────────────────────┐
│  piko cc notify                                                │
│  - Reads hook JSON from stdin                                  │
│  - POSTs to server /api/orchestra/notify                       │
│  - Exits immediately                                           │
└───────────────────────────┬───────────────────────────────────┘
                            │ HTTP POST
                            ▼
┌───────────────────────────────────────────────────────────────┐
│                      Piko Server                               │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  Orchestra Hub                                           │  │
│  │  - WebSocket connections from UI clients                 │  │
│  │  - Pending notifications store                           │  │
│  │  - Broadcasts to all clients                             │  │
│  └─────────────────────────────────────────────────────────┘  │
└───────────────────────────┬───────────────────────────────────┘
                            │ WebSocket
                            ▼
┌───────────────────────────────────────────────────────────────┐
│                       Browser UI                               │
│  - Shows notification badge/toast                              │
│  - User can respond → POST /api/orchestra/respond              │
│  - Server uses tmux send-keys to inject response               │
└───────────────────────────────────────────────────────────────┘
```

---

## Milestones

### Milestone 1: WebSocket Infrastructure
Add WebSocket support to the server with a hub for managing connections.

- [x] Add `github.com/gorilla/websocket` to go.mod
- [x] Create `internal/server/orchestra.go` with message types
- [x] Implement Hub struct (client registry, broadcast channel)
- [x] Implement Client struct (connection, send channel, read/write pumps)
- [x] Add WebSocket upgrade handler
- [x] Register `GET /api/orchestra/ws` route in server.go
- [x] Add pending notifications in-memory store

---

### Milestone 2: Notification API
Add endpoints for Claude Code hooks to send notifications and for UI to respond.

- [x] Add `POST /api/orchestra/notify` handler
- [x] Add `POST /api/orchestra/respond` handler
- [x] Add `GET /api/orchestra/notifications` handler (list pending)
- [x] Add `DELETE /api/orchestra/notifications/{id}` handler (dismiss)
- [x] Add `SendKeysToSession(sessionName, keys)` to tmux package
- [x] Wire response handler to call tmux send-keys
- [x] Broadcast notifications to WebSocket clients

---

### Milestone 3: CLI Commands
Add CLI commands for Claude Code hooks and manual responses.

- [x] Create `internal/cli/cc.go` - parent `piko cc` command
- [x] Create `internal/cli/cc_notify.go` - `piko cc notify` command
  - [x] Read hook JSON from stdin
  - [x] Extract notification_type, message
  - [x] Detect env from $PIKO_ENV_NAME or cwd
  - [x] POST to server
- [x] Create `internal/cli/respond.go` - `piko respond` command
  - [x] Interactive picker when no args
  - [x] Direct response when args provided
- [x] Register commands in root.go

---

### Milestone 4: UI Integration
Add WebSocket client and notification UI to the web interface.

- [x] Add WebSocket connection manager (connect, reconnect)
- [x] Add connection status indicator in header
- [x] Add notification badge (count of pending)
- [x] Add notification panel/dropdown
- [x] Add response modal (input + send)
- [x] Handle incoming WebSocket messages
- [x] Wire response to POST /api/orchestra/respond

---

### Milestone 5: Claude Code Hook Configuration
Create hook configuration generator.

- [x] Create `internal/cli/cc_init.go` - `piko cc init` command
- [x] Generate `.claude/settings.json` with Notification hook
- [x] Set hook to call `piko cc notify`

---

## File Summary

| File | Action | Milestone |
|------|--------|-----------|
| `go.mod` | Modify | 1 |
| `internal/server/orchestra.go` | Create | 1, 2 |
| `internal/server/server.go` | Modify | 1, 2 |
| `internal/tmux/session.go` | Modify | 2 |
| `internal/cli/cc.go` | Create | 3 |
| `internal/cli/cc_notify.go` | Create | 3 |
| `internal/cli/respond.go` | Create | 3 |
| `internal/cli/root.go` | Modify | 3 |
| `internal/cli/cc_init.go` | Create | 5 |
| `internal/server/static/index.html` | Modify | 4 |

---

## Verification

1. **WebSocket Connection**: Start server, open UI, check console for connection
2. **Notification Flow**: Simulate notification via CLI, verify UI shows it
3. **Response Flow**: Respond via UI, verify tmux receives keystrokes
4. **End-to-End**: Run `piko cc init`, use Claude Code, verify full flow
