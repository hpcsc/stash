# WebSocket

Stash uses WebSockets to push real-time updates from the server to the web
client — for example, when the link-metadata reactor enriches a captured link
with its page title and description.

## Architecture

```
Browser                          Server (cmd/api)
  │                                  │
  │  GET /ws?token=... ──upgrade──►  │  websocket.Accept()
  │                                  │
  │◄──── {type:"link-metadata",  ─── │  hub.Broadcast()
  │       data:{id,title,desc}}      │
  │                                  │
  │  ping ───────────────────────►   │  (handled by library)
  │◄────────────── pong ──────────── │
```

The API process (`cmd/api`) hosts the WebSocket endpoint. It subscribes to
domain events via PostgreSQL `LISTEN`/`NOTIFY` (same mechanism as `cmd/catchup`)
and forwards relevant events to connected clients.

The reactor itself still runs in `cmd/catchup` — the API only forwards the
results, it doesn't do the work.

## Setup (Go)

### Dependencies

```
go get github.com/coder/websocket
```

`coder/websocket` is a modern, minimal, context-aware WebSocket library. It is
the actively-maintained fork of `nhooyr.io/websocket`. Prefer it over
`gorilla/websocket` for new projects.

### Hub

The hub groups connected clients by user and broadcasts only to the user an
event belongs to:

```go
type Hub struct {
    mu      sync.RWMutex
    clients map[domain.UserID]map[*websocket.Conn]struct{}
}

func (h *Hub) Register(user domain.UserID, conn *websocket.Conn) {
    h.mu.Lock()
    if h.clients[user] == nil {
        h.clients[user] = make(map[*websocket.Conn]struct{})
    }
    h.clients[user][conn] = struct{}{}
    h.mu.Unlock()
}

func (h *Hub) Unregister(user domain.UserID, conn *websocket.Conn) {
    h.mu.Lock()
    delete(h.clients[user], conn)
    h.mu.Unlock()
}

func (h *Hub) Broadcast(ctx context.Context, user domain.UserID, msg []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for conn := range h.clients[user] {
        conn.Write(ctx, websocket.MessageText, msg)
    }
}
```

The push subscription derives the user from the event stream (`domain.UserOf`),
so one user's link or thumbnail update never reaches another user's tab.

The hub is meant for single-instance, low-connection-count deployments. For
production at scale, see "Scaling in production" below.

### Upgrade handler

```go
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Auth: read token from query param (WebSocket upgrade
    // happens before middleware can read the Authorization header)
    token := r.URL.Query().Get("token")
    subject, err := issuer.Verify(token)
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    user := domain.UserID(subject)
    if !user.Valid() {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        InsecureSkipVerify: true,
    })
    if err != nil {
        return
    }
    defer conn.Close(websocket.StatusNormalClosure, "")

    h.Register(user, conn)
    defer h.Unregister(user, conn)

    // Read loop — keeps the connection alive, handles close
    for {
        _, _, err := conn.Read(r.Context())
        if err != nil {
            break
        }
    }
}
```

### Registration

Register the WebSocket route as a protected endpoint alongside other routes:

```go
wsHub := websocket.NewHub()
wsHandler := websocket.NewHandler(wsHub, issuer)

// Protected route — auth via query param token in the handler itself,
// since WebSocket upgrade bypasses HTTP middleware
routables := []route.Routable{
    // ... existing routables
    wsHandler,
}
```

## Connection lifecycle

```
Client connects ─► token validated ─► registered in hub ─► read-loop started
                                                                    │
                                             ┌──────────────────────┘
                                             │
                              client disconnects / read error
                                             │
                                    unregistered from hub
                                    connection closed
```

### Authentication

WebSocket connections authenticate via a query parameter (`?token=...`) rather
than the `Authorization` header. This is because the WebSocket upgrade handshake
is an HTTP GET request, and browsers' `WebSocket` constructor doesn't support
custom headers.

The token is the same JWT issued at sign-in. The handler validates it before
accepting the upgrade. An invalid token returns HTTP 401 before the upgrade
happens.

### Heartbeat / keepalive

`coder/websocket` handles ping/pong automatically. The server sends pings; the
browser responds with pongs. If the browser doesn't respond within the read
deadline, the connection is closed.

Configure on the server:
```go
conn.CloseRead(ctx) // sets a read deadline; library sends pings
```

On the browser, no configuration needed — the browser's `WebSocket`
implementation handles pong responses automatically.

### Graceful shutdown

On SIGTERM:

1. Stop accepting new connections (close the listener).
2. Send close frames to all connected clients with a 5-second deadline.
3. Wait for existing HTTP requests to finish.
4. Exit.

The hub's `Shutdown` method:
```go
func (h *Hub) Shutdown(ctx context.Context) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for conn := range h.clients {
        conn.Close(websocket.StatusGoingAway, "server shutting down")
    }
}
```

## Client (browser)

```ts
const ws = new WebSocket(`ws://localhost:3333/ws?token=${token}`)

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)
  switch (msg.type) {
    case 'link-metadata-fetched':
      // update the matching item in local state
      break
  }
}

ws.onclose = () => {
  // reconnect with exponential backoff
  setTimeout(connect, backoff)
}
```

No library needed — the `WebSocket` API is built into every browser.

### Reconnection

On disconnect, reconnect with exponential backoff:

```ts
let backoff = 1000
const maxBackoff = 30000

function connect() {
  const ws = new WebSocket(url)
  ws.onclose = () => {
    setTimeout(connect, backoff)
    backoff = Math.min(backoff * 2, maxBackoff)
  }
  ws.onopen = () => { backoff = 1000 }
}
```

## Scaling in production

The design above works for a single instance with dozens to low hundreds of
connections. For thousands of concurrent users, several things change.

### File descriptor limits

Each WebSocket connection consumes one file descriptor. The default `ulimit -n`
on Linux is 1024. Raise it:

```bash
# /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536
```

### Memory per connection

Each Go WebSocket connection while idle:

| Resource | Approximate size |
|---|---|
| Goroutine stack | 2–8 KB (grows as needed) |
| Read buffer | 4 KB |
| Write buffer | 4 KB |
| Connection struct | ~1 KB |

Total: ~11–17 KB per idle connection. At 100,000 connections: ~1.5 GB. A single
machine can handle this.

### Multi-instance fan-out (horizontal scaling)

When connections are spread across multiple server instances, a broadcast must
reach clients on every instance. The hub-per-instance model isn't enough.

Use a pub/sub layer (Redis, NATS) between instances:

```
                    ┌──────────┐
  event arrives ──► │  Redis   │ (publish)
                    └────┬─────┘
            ┌────────────┼────────────┐
            ▼            ▼            ▼
        ┌──────┐    ┌──────┐    ┌──────┐
        │ WS 1 │    │ WS 2 │    │ WS 3 │
        └──┬───┘    └──┬───┘    └──┬───┘
           ▼           ▼           ▼
        clients      clients      clients
```

Each instance subscribes to a Redis channel. When it receives a message to
broadcast, it publishes to Redis. Every instance, including the publisher,
receives it and forwards to local WebSocket connections.

With Go, `rueidis` is the recommended Redis client (non-blocking, pipelined).

### Goroutine-per-connection vs event loop

Go's goroutine-per-connection model works well up to ~100k connections. Past
that, goroutine stack memory becomes a bottleneck. At that scale, switch to
an event-loop model:

- `gnet` — high-performance event-loop networking for Go
- `evio` — similar, lighter

Both use epoll/kqueue under the hood and avoid per-connection goroutines.

In practice, most apps never hit this ceiling. A single instance with
goroutine-per-connection handles 50k concurrent users comfortably.

### Connection draining on deploy

When deploying a new version:

1. New instances start and register with the load balancer.
2. Old instances receive SIGTERM.
3. Old instances stop accepting new connections.
4. Old instances send close frames to all connected clients (status 1001,
   "going away").
5. Clients reconnect to the new instances (exponential backoff in the client).
6. Old instances shut down after a grace period (30 seconds).

## Message format

All messages are JSON with a `type` field:

```json
{
  "type": "link-metadata-fetched",
  "data": {
    "id": "uuid",
    "title": "Page Title",
    "description": "Meta description from the page"
  }
}
```

The client switches on `type` to route the message. New message types are
added as new reactors or events need to push updates.

## Best practices

### Don't put tokens in URL query strings in production

The token in `?token=...` is visible in server logs, proxy logs, and browser
history. For production, prefer one of:

- **Cookie:** set an `HttpOnly` cookie at sign-in; the browser sends it
  automatically on the WebSocket upgrade request.
- **First-message auth:** accept the WebSocket upgrade unauthenticated, then
  require the first message from the client to be an auth message. Close the
  connection if it doesn't arrive within 5 seconds.

Stash uses the query-param approach because it is self-hosted and the token
scopes both files and realtime updates to one user — the risk is negligible.

### Keep messages small

A single large message blocks all other messages on that connection. If you
need to send bulk data, chunk it or use a separate HTTP request.

### One connection per client

Don't open multiple WebSocket connections from the same browser tab. Use one
connection and multiplex message types.

### Timeouts

Set a read deadline on the server. If the client stops responding (tab
backgrounded, network loss), the connection is cleaned up promptly rather than
leaking.

`coder/websocket` handles this when you call `conn.CloseRead(ctx)`:

```go
conn.CloseRead(ctx) // sets read deadline from context
```

### Logging

Log connects, disconnects, and errors at `INFO` or `WARN` level. Don't log
at `ERROR` for normal disconnects — they're expected.

## References

- [coder/websocket](https://github.com/coder/websocket)
- [MDN: WebSocket API](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
- [Go net/http: Hijack](https://pkg.go.dev/net/http#Hijacker) (low-level
  alternative, not needed with coder/websocket)