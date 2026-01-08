# Milestone 5: Web UI

**Priority:** Medium
**Depends on:** M1-M4 (all CLI functionality)
**Unlocks:** Visual management, easier port discovery

## Goal

Browser-based UI at `localhost:19876` for managing environments. Provides at-a-glance port visibility and one-click actions.

## Success Criteria

```bash
$ piko server
→ Piko server running at http://localhost:19876
  Press Ctrl+C to stop

# Browser shows:
# - All environments with status
# - Port mappings for each
# - Buttons: Open in Cursor, Start, Stop, Logs
```

## Tasks

### 5.1 HTTP Server
- [ ] Create server on port 19876
- [ ] Serve static files for UI
- [ ] Implement API endpoints
- [ ] Graceful shutdown on Ctrl+C

### 5.2 API Endpoints
```
GET  /api/project
  → { "name": "myapp", "path": "/path/to/project" }

GET  /api/environments
  → [{ "name": "feature-auth", "status": "running", "branch": "...", "ports": [...] }]

GET  /api/environments/:name
  → { "name": "...", "status": "...", "containers": [...], "ports": [...] }

POST /api/environments/:name/up
  → { "success": true }

POST /api/environments/:name/down
  → { "success": true }

POST /api/environments/:name/open
  → { "success": true }  // Opens in Cursor

GET  /api/environments/:name/logs?follow=true
  → Server-Sent Events stream of logs
```

### 5.3 Static UI
- [ ] Single HTML file with embedded CSS/JS
- [ ] No build step, no npm
- [ ] Vanilla JS (no framework)
- [ ] Embed in Go binary using `embed` package

### 5.4 UI Components
- [ ] Project header with name
- [ ] Environment cards:
  - Name, status indicator (●/○)
  - Branch name
  - Port list with clickable URLs
  - Action buttons
- [ ] Auto-refresh every 5 seconds (or SSE)
- [ ] Manual refresh button

### 5.5 Actions
- [ ] "Open in Cursor" → POST /api/environments/:name/open
- [ ] "Start" → POST /api/environments/:name/up
- [ ] "Stop" → POST /api/environments/:name/down
- [ ] "View Logs" → Opens modal with streaming logs
- [ ] Port URLs → Direct links (open in new tab)

### 5.6 Embed in Binary
- [ ] Use `//go:embed` directive
- [ ] Serve from embedded filesystem
- [ ] Single binary, no external files

### 5.7 Server Command
- [ ] `piko server` — start on default port
- [ ] `piko server --port 8080` — custom port
- [ ] Print URL on startup
- [ ] Auto-open browser (optional, `--open` flag)

## UI Mockup

```html
┌─────────────────────────────────────────────────────────────────┐
│  piko                                          myapp   [↻]     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ● feature-auth                              running     │   │
│  │   branch: feature-auth                                  │   │
│  │                                                         │   │
│  │   app    http://localhost:52341  [↗]                    │   │
│  │   db     localhost:52342                                │   │
│  │   redis  localhost:52343                                │   │
│  │                                                         │   │
│  │   [Open in Cursor]  [Logs]  [Stop]                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ○ feature-payments                          stopped     │   │
│  │   branch: feature-payments                              │   │
│  │                                                         │   │
│  │   [Open in Cursor]  [Start]                             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Choices

- **Server**: Go `net/http` (no framework needed)
- **HTML**: Single file, semantic HTML
- **CSS**: Minimal, embedded in `<style>` tag
- **JS**: Vanilla, fetch API, no build step
- **Embedding**: Go 1.16+ `embed` package

## Test Cases

1. **Server starts**: Listens on 19876
2. **GET /api/environments**: Returns list
3. **POST up/down**: Starts/stops containers
4. **POST open**: Opens Cursor
5. **UI loads**: Shows environments
6. **UI refresh**: Updates status
7. **Port links**: Clickable, open in new tab
8. **Logs modal**: Streams logs
9. **Error handling**: Shows errors in UI

## Definition of Done

- [ ] `piko server` starts HTTP server
- [ ] API returns environment data
- [ ] UI shows all environments
- [ ] UI shows port mappings
- [ ] "Open in Cursor" works
- [ ] Start/Stop buttons work
- [ ] Logs viewable in browser
- [ ] Auto-refresh works
- [ ] Single binary (no external files)
