# Streaming Output: Real-time Feedback for Client Operations

## Problem

When a user runs `piko create` while the piko server is running, the client sends a request to the server and waits for a response. The server executes potentially long-running operations (git clone, npm install, cargo build, docker compose up), but the client displays nothing until everything completes.

This creates a poor user experience:

- User sees a hanging terminal with no feedback
- No way to know if progress is being made or if something is stuck
- Server logs show activity, but the user can't see them
- For operations taking 30+ seconds, users may think piko is broken

## Goals

1. Stream server-side output to the client in real-time
2. Maintain existing fallback behavior when server is down (local execution works as-is)
3. Provide structured messages so clients can format output appropriately
4. Handle connection failures gracefully without losing work

## Non-Goals

- Bidirectional streaming (client sending input to server) - not needed for v1
- Streaming for all operations - focus on `create` first, extend later
- Web UI streaming - terminal client is the priority

## Design

### Protocol Choice: WebSocket

WebSocket is preferred over chunked HTTP or SSE because:

- Piko already uses `gorilla/websocket` for other features
- Structured message format allows distinguishing log sources
- Clean separation between streaming logs and final result
- Easier to handle connection state and errors
- Can extend to bidirectional communication later if needed

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Server                                                      │
│                                                             │
│   WebSocket Handler (/api/environments/create/stream)       │
│       │                                                     │
│       ▼                                                     │
│   StreamWriter (implements io.Writer)                       │
│       │                                                     │
│       ├──► WebSocket connection (to client)                 │
│       └──► os.Stdout (local server logs)                    │
│                                                             │
│   operations.CreateEnvironment(ctx, opts, writer)           │
│       │                                                     │
│       ├── git worktree ──► StreamWriter                     │
│       ├── docker up    ──► StreamWriter                     │
│       └── scripts      ──► StreamWriter                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
         │
         │ WebSocket messages (real-time)
         ▼
┌─────────────────────────────────────────────────────────────┐
│ Client                                                      │
│                                                             │
│   1. Connect to WebSocket                                   │
│   2. Send create request                                    │
│   3. Read messages, print to terminal                       │
│   4. Handle final "complete" or "error" message             │
│   5. Close connection                                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Message Protocol

#### Client → Server

Single message to initiate the create operation:

```json
{
  "action": "create",
  "project": "myproject",
  "environment": "feature-xyz",
  "attach": true
}
```

#### Server → Client

**Log messages** (streaming, multiple):

```json
{
  "type": "log",
  "source": "git",
  "stream": "stdout",
  "data": "Cloning into '.piko/worktrees/feature-xyz'...\n"
}
```

```json
{
  "type": "log",
  "source": "script:setup",
  "stream": "stderr",
  "data": "npm WARN deprecated package@1.0.0\n"
}
```

**Completion message** (final, exactly one):

```json
{
  "type": "complete",
  "success": true,
  "environment": {
    "id": 42,
    "name": "feature-xyz",
    "project": "myproject",
    "mode": "docker",
    "status": "running"
  }
}
```

**Error message** (final, instead of complete):

```json
{
  "type": "complete",
  "success": false,
  "error": "npm install failed with exit code 1"
}
```

### Source Identifiers

| Source           | Description                                         |
| ---------------- | --------------------------------------------------- |
| `git`            | Git operations (worktree create, clone, fetch)      |
| `docker`         | Docker compose operations                           |
| `script:prepare` | User-defined prepare script                         |
| `script:setup`   | User-defined setup script                           |
| `piko`           | Internal piko messages (creating directories, etc.) |

---

## Server-Side Implementation

### 1. StreamWriter Component

**Location:** `internal/stream/writer.go` (new file)

**Responsibilities:**

- Implement `io.Writer` interface
- Marshal writes into JSON messages
- Send over WebSocket connection
- Optionally tee to local stdout for server-side logging
- Handle write errors gracefully (don't crash if connection drops)
- Thread-safe for concurrent writes from multiple goroutines

**Behavior:**

- Line-buffered: accumulate bytes until newline, then send as one message
- Timeout on slow consumers: if WebSocket send blocks >5s, drop message and continue
- Source tagging: each StreamWriter instance is created with a source label

**Interface:**

```go
type StreamWriter struct {
    conn   *websocket.Conn
    source string           // e.g., "git", "script:setup"
    stream string           // "stdout" or "stderr"
    mu     sync.Mutex       // protects conn writes
    buf    bytes.Buffer     // line buffering
}

func NewStreamWriter(conn *websocket.Conn, source, stream string) *StreamWriter
func (w *StreamWriter) Write(p []byte) (n int, err error)
func (w *StreamWriter) Flush() error  // send any buffered content
```

### 2. Modify CreateEnvironment

**Location:** `internal/operations/environment.go`

**Changes:**

- Add optional `io.Writer` parameter (or use functional options pattern)
- If writer is nil, fall back to current behavior (os.Stdout)
- Pass writer to all sub-operations

**Signature change:**

```go
// Before
func CreateEnvironment(ctx context.Context, opts CreateOpts) (*Environment, error)

// After
func CreateEnvironment(ctx context.Context, opts CreateOpts, output io.Writer) (*Environment, error)
```

**Propagation points:**

- `ScriptRunner.Run()` - set cmd.Stdout/cmd.Stderr to writer
- `git.CreateWorktree()` - replace CombinedOutput with streaming
- `docker.ComposeUp()` - replace Run() with streaming to writer

### 3. Modify ScriptRunner

**Location:** `internal/config/runner.go`

**Changes:**

- Accept writer parameter
- Create separate StreamWriters for stdout and stderr with appropriate source labels

### 4. Modify Git Operations

**Location:** `internal/git/worktree.go`

**Changes:**

- Replace `cmd.CombinedOutput()` with:
  ```go
  cmd.Stdout = writer
  cmd.Stderr = writer
  err := cmd.Run()
  ```
- Pass writer through from CreateEnvironment

### 5. Modify Docker Operations

**Location:** `internal/docker/compose.go` (or similar)

**Changes:**

- Same pattern as git: replace buffered output with streaming
- Docker compose can be verbose; consider filtering or prefixing lines

### 6. WebSocket Endpoint

**Location:** `internal/api/handlers.go` (or new file `internal/api/stream.go`)

**New endpoint:** `GET /api/environments/create/stream`

**Handler flow:**

1. Upgrade HTTP connection to WebSocket
2. Read initial message (create request)
3. Validate request parameters
4. Create StreamWriter instances for each source
5. Call `operations.CreateEnvironment()` with writers
6. Send final complete message (success or error)
7. Close WebSocket connection

**Error handling:**

- If WebSocket drops mid-operation, continue the create (don't leave half-created env)
- Log that client disconnected
- Operation result is still valid, just not streamed

### 7. Context Cancellation (Optional for v1)

If client disconnects and we want to cancel:

- Pass context from WebSocket handler to CreateEnvironment
- Check context.Done() at stage boundaries
- Clean up partial state on cancellation

For v1, recommend: let operation complete even if client disconnects.

---

## Client-Side Implementation

### 1. WebSocket Client

**Location:** `internal/api/stream_client.go` (new file)

**Responsibilities:**

- Establish WebSocket connection to server
- Send create request
- Read messages in a loop
- Dispatch messages to appropriate handler (log → print, complete → return)
- Handle connection errors

### 2. Output Formatting

**Location:** `internal/cli/create.go`

Output is displayed with source prefixes for context:

```
[git] Cloning into '.piko/worktrees/feature-xyz'...
[git] remote: Enumerating objects: 1234, done.
[setup] npm WARN deprecated package@1.0.0
[setup] added 847 packages in 12s
```

The prefix is derived from the `source` field in log messages. Stderr output can optionally be displayed in a different color (e.g., yellow) to distinguish warnings/errors from normal output.

### 3. Fallback Behavior

**Location:** `internal/cli/create.go`

```
piko create feature-xyz
    │
    ├── Server running?
    │   └── Yes → Connect WebSocket, stream output
    │
    └── Server not running?
        └── operations.CreateEnvironment() locally (already streams to stdout)
```

The local fallback already works because scripts write directly to os.Stdout. If WebSocket connection fails, fall back to local execution.

---

## Edge Cases

| Scenario                            | Handling                                                   |
| ----------------------------------- | ---------------------------------------------------------- |
| Client disconnects mid-stream       | Server completes operation, logs disconnection             |
| Server crashes mid-operation        | Client sees connection drop, shows error                   |
| Very long operation (>5 min)        | WebSocket ping/pong keeps connection alive                 |
| Binary/non-UTF8 output              | Base64 encode in message, or filter to printable           |
| Concurrent creates from same client | Each gets own WebSocket connection                         |
| Rate limiting (very fast output)    | Client-side: just print. Server-side: line buffering helps |

---

## Implementation Phases

### Phase 1: Core Infrastructure

- [ ] Create `StreamWriter` component
- [ ] Add WebSocket endpoint
- [ ] Modify `CreateEnvironment` to accept writer parameter
- [ ] Wire through to `ScriptRunner` (scripts are most visible)

### Phase 2: Full Streaming

- [ ] Modify git operations to use streaming
- [ ] Modify docker operations to use streaming
- [ ] Implement complete/error messages

### Phase 3: Client Integration

- [ ] Create WebSocket client in API package
- [ ] Modify `piko create` to use streaming
- [ ] Add output formatting (prefixed mode)
- [ ] Handle fallback to local execution

---

## Testing Strategy

1. **Unit tests:** StreamWriter correctly buffers and sends messages
2. **Integration tests:** WebSocket endpoint receives request, streams output, sends completion
3. **Manual testing:** Run `piko create` with server running, verify output appears in real-time
4. **Fallback testing:** Stop server, verify local execution still works
5. **Disconnect testing:** Kill client mid-create, verify server completes without error

---

## Estimated Effort

| Component                          | Effort         |
| ---------------------------------- | -------------- |
| StreamWriter                       | 1-2 hours      |
| WebSocket endpoint                 | 2-3 hours      |
| Modify CreateEnvironment + sub-ops | 2-3 hours      |
| Client WebSocket handling          | 2-3 hours      |
| Output formatting (prefixed)       | 1 hour         |
| Testing + edge cases               | 2 hours        |
| **Total**                          | **9-14 hours** |
