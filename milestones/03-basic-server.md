# Milestone 3: Basic Server

**Priority:** High
**Depends on:** M1 (Init & Create), M2 (Lifecycle)
**Unlocks:** Browser-based workflow, M6 (Full Web UI)

## Goal

HTTP server with basic UI to list environments, create new ones, and open in Cursor. Enables browser-first workflow early.

## Success Criteria

```bash
$ piko server
→ Piko server running at http://localhost:19876

# Browser shows:
# - List of environments
# - "Create New" button
# - "Open in Cursor" per environment
```

## Tasks

### 3.1 HTTP Server
- [x] Create server on port 19876
- [x] `piko server` command
- [x] `piko server --port 8080` — custom port
- [x] Graceful shutdown on Ctrl+C
- [x] Print URL on startup

### 3.2 API Endpoints
```
GET  /api/project
  → { "name": "myapp", "path": "/path/to/project", "initialized": true }

GET  /api/environments
  → [{ "name": "feature-auth", "status": "running", "branch": "..." }]

POST /api/environments
  Body: { "name": "feature-x", "branch": "feature-x" }
  → { "success": true, "environment": {...} }

POST /api/environments/:name/open
  → { "success": true }  // Opens in Cursor
```

### 3.3 Create from Browser
- [x] POST /api/environments creates new environment
- [x] Calls same logic as `piko create`
- [x] Returns success/error
- [x] UI shows progress/result

### 3.4 Open in Cursor
- [x] POST /api/environments/:name/open
- [x] Runs: `cursor <worktree-path>`
- [x] Falls back: `code <path>`, `$EDITOR <path>`
- [x] Returns success/error

### 3.5 Static UI
- [x] Single HTML file with embedded CSS/JS
- [x] No build step, no npm
- [x] Vanilla JS (fetch API)
- [x] Embed in Go binary using `embed` package

### 3.6 UI Components
- [x] Project header with name
- [x] Environment list:
  - Name, status indicator (●/○)
  - Branch name
  - "Open in Cursor" button
- [x] "Create New" button/form:
  - Name input
  - Branch input (optional, defaults to name)
  - Submit button
- [x] Simple error display

### 3.7 Embed in Binary
- [x] Use `//go:embed` directive
- [x] Serve from embedded filesystem
- [x] Single binary, no external files

## UI Mockup

```
┌─────────────────────────────────────────────────────────────────┐
│  piko                                              myapp        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [+ Create New Environment]                                     │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ● feature-auth                              running      │   │
│  │   branch: feature-auth                                   │   │
│  │   [Open in Cursor]                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ ○ feature-payments                          stopped      │   │
│  │   branch: feature-payments                               │   │
│  │   [Open in Cursor]                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Create New Dialog:
┌─────────────────────────────────────────────────────────────────┐
│  Create New Environment                                    [x]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Name:   [feature-xyz____________]                              │
│  Branch: [feature-xyz____________] (optional)                   │
│                                                                 │
│  [Cancel]                                        [Create]       │
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
2. **GET /api/project**: Returns project info
3. **GET /api/environments**: Returns list
4. **POST /api/environments**: Creates new environment
5. **POST open**: Opens Cursor
6. **UI loads**: Shows environment list
7. **Create from UI**: Form works, creates environment
8. **Open from UI**: Opens Cursor
9. **Error handling**: Shows errors in UI

## Definition of Done

- [x] `piko server` starts HTTP server
- [x] API returns project and environment data
- [x] API can create new environments
- [x] API can trigger "Open in Cursor"
- [x] UI shows all environments
- [x] UI has "Create New" form
- [x] UI has "Open in Cursor" button
- [x] Single binary (no external files)
