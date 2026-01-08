# Milestone 6: Full Web UI

**Priority:** Medium
**Depends on:** M3 (Basic Server), M5 (Inspection)
**Unlocks:** Complete browser-based management

## Goal

Enhance the basic server UI with port display, start/stop controls, logs streaming, and auto-refresh. Full environment management from browser.

## Success Criteria

```bash
$ piko server
→ Piko server running at http://localhost:19876

# Browser shows (building on M3):
# - Port mappings per environment
# - Start/Stop buttons
# - Logs streaming
# - Auto-refresh
```

## Builds on M3

M3 (Basic Server) provides:
- HTTP server on :19876
- List environments
- Create new environments
- Open in Cursor

M6 adds:
- Port display
- Start/Stop controls
- Logs streaming
- Auto-refresh
- Detailed status

## Tasks

### 6.1 Enhanced API Endpoints
```
GET  /api/environments/:name
  → { "name": "...", "status": "...", "containers": [...], "ports": [...] }

POST /api/environments/:name/up
  → { "success": true }

POST /api/environments/:name/down
  → { "success": true }

GET  /api/environments/:name/logs?follow=true
  → Server-Sent Events stream of logs
```

### 6.2 Port Display
- [ ] Fetch port mappings from Docker
- [ ] Display in environment cards
- [ ] Clickable URLs for HTTP ports
- [ ] Copy-to-clipboard for connection strings

### 6.3 Start/Stop Controls
- [ ] "Start" button for stopped environments
- [ ] "Stop" button for running environments
- [ ] Show loading state during operation
- [ ] Update status after completion

### 6.4 Logs Streaming
- [ ] SSE endpoint for log streaming
- [ ] Logs modal/panel in UI
- [ ] Service filter (all or specific)
- [ ] Follow mode (auto-scroll)
- [ ] Stop/disconnect button

### 6.5 Auto-Refresh
- [ ] Poll every 5 seconds for status updates
- [ ] Or: SSE for real-time updates
- [ ] Manual refresh button
- [ ] Show "last updated" timestamp

### 6.6 Enhanced UI Components
- [ ] Environment cards with ports:
  ```
  ● feature-auth                              running
    branch: feature-auth

    app    http://localhost:52341  [↗]
    db     localhost:52342         [copy]
    redis  localhost:52343         [copy]

    [Open in Cursor]  [Logs]  [Stop]
  ```
- [ ] Status indicators with health info
- [ ] Container-level status (X/Y healthy)

## UI Mockup

```
┌─────────────────────────────────────────────────────────────────┐
│  piko                                          myapp   [↻]     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [+ Create New Environment]                                     │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ● feature-auth                    running (3/3 healthy) │   │
│  │   branch: feature-auth                                   │   │
│  │                                                         │   │
│  │   app    http://localhost:52341  [↗]                    │   │
│  │   db     localhost:52342         [📋]                   │   │
│  │   redis  localhost:52343         [📋]                   │   │
│  │                                                         │   │
│  │   [Open in Cursor]  [Logs]  [Stop]                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ○ feature-payments                          stopped     │   │
│  │   branch: feature-payments                               │   │
│  │                                                         │   │
│  │   [Open in Cursor]  [Start]  [Destroy]                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Test Cases

1. **Port display**: Shows correct ports
2. **Port links**: HTTP ports clickable
3. **Copy button**: Copies connection string
4. **Start button**: Starts stopped environment
5. **Stop button**: Stops running environment
6. **Logs stream**: Shows live logs
7. **Logs filter**: Can filter by service
8. **Auto-refresh**: Status updates automatically
9. **Error handling**: Shows errors in UI

## Definition of Done

- [ ] Ports displayed per environment
- [ ] HTTP ports are clickable links
- [ ] Copy-to-clipboard for connection strings
- [ ] Start/Stop buttons work
- [ ] Logs viewable in browser
- [ ] Logs streaming via SSE
- [ ] Auto-refresh works
- [ ] Health status shown
