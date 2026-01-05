# Shop Messages Real-Time WebSocket Design

## Executive Summary

This document outlines the design and implementation plan for adding real-time messaging capabilities to the Shop Messages feature using WebSocket connections. This will eliminate the need for mobile clients to constantly poll for new messages, reducing server load and improving user experience.

**Design Philosophy**: Simple, maintainable, and production-ready implementation following the "Simplicity (Recommended)" approach selected by the user.

**Status**: Design phase - Ready for implementation
**Created**: 2026-01-03
**Author**: swisscheese

---

## Table of Contents

1. [Current Implementation Analysis](#current-implementation-analysis)
2. [Problem Statement](#problem-statement)
3. [Proposed Solution](#proposed-solution)
4. [Architecture Design](#architecture-design)
5. [Technical Implementation](#technical-implementation)
6. [API Specification](#api-specification)
7. [Security Considerations](#security-considerations)
8. [Deployment Strategy](#deployment-strategy)
9. [Testing Strategy](#testing-strategy)
10. [Performance Considerations](#performance-considerations)
11. [References](#references)

---

## Current Implementation Analysis

### Existing Endpoints

**Current shop message endpoints** ([shops_route.go:47-54](api/route/shops_route.go#L47-L54)):
```
POST   /api/v1/shops/messages                      - CreateShopMessage
GET    /api/v1/shops/:shop_id/messages             - GetShopMessages
GET    /api/v1/shops/:shop_id/messages/paginated   - GetShopMessagesPaginated
PUT    /api/v1/shops/messages                      - UpdateShopMessage
DELETE /api/v1/shops/messages/:message_id          - DeleteShopMessage
POST   /api/v1/shops/messages/image/upload         - UploadMessageImage
DELETE /api/v1/shops/messages/image/:message_id    - DeleteMessageImage
```

### Current Data Model

**ShopMessages structure** ([.gen/miltech_ng/public/model/shop_messages.go]()):
```go
type ShopMessages struct {
    ID        string     `json:"id"`
    ShopID    string     `json:"shop_id"`
    UserID    string     `json:"user_id"`
    Message   string     `json:"message"`
    CreatedAt *time.Time `json:"created_at"`
    UpdatedAt *time.Time `json:"updated_at"`
    IsEdited  *bool      `json:"is_edited"`
}
```

### Current Architecture

The existing implementation follows a clean architecture pattern:
- **Controller Layer**: [shops_controller.go](api/controller/shops_controller.go) - HTTP request handling
- **Service Layer**: [shops_service_impl.go](api/service/shops_service_impl.go) - Business logic
- **Repository Layer**: [shops_repository_impl.go](api/repository/shops_repository_impl.go) - Database operations
- **Route Layer**: [shops_route.go](api/route/shops_route.go) - Endpoint registration

### Current Authorization

All endpoints verify:
1. User authentication via middleware (JWT token)
2. Shop membership via repository checks
3. Permission levels for certain operations (admin/member)

---

## Problem Statement

### Current Pain Points

1. **Inefficient polling**: Mobile clients must repeatedly query `GET /shops/:shop_id/messages` to check for new messages
2. **Server load**: Constant polling creates unnecessary database queries and network traffic
3. **Battery drain**: Frequent HTTP requests consume mobile device battery
4. **Delayed updates**: Users only see new messages when their polling interval triggers
5. **Bandwidth waste**: Polling often returns empty results when no new messages exist

### User Experience Impact

- **Latency**: Messages can take several seconds to appear (depending on poll interval)
- **Inconsistency**: Different clients may poll at different rates
- **Resource usage**: Unnecessary network activity even when chat is inactive

---

## Proposed Solution

### WebSocket Real-Time Messaging

Implement a WebSocket endpoint that allows mobile clients to:
1. **Connect** to a shop-specific message stream
2. **Receive** new messages in real-time as they're sent
3. **Reconnect** gracefully on disconnect with simple recovery
4. **Fallback** to existing polling endpoints if WebSocket fails

### Design Decisions (Based on User Preferences)

✅ **Simplicity Priority**: Single-server implementation without external dependencies (no Redis)
✅ **Message Scope**: Only deliver messages sent AFTER client connects
✅ **Disconnect Handling**: Simple reconnect - client polls API for missed messages

### Why WebSocket (vs Alternatives)

**WebSocket** is the recommended approach because:

✅ **Bi-directional**: Full-duplex communication (though we primarily use server→client)
✅ **Efficient**: Single persistent connection vs repeated HTTP requests
✅ **Low latency**: Messages delivered instantly when sent
✅ **Framework compatible**: Works seamlessly with Gin via gorilla/websocket
✅ **Mobile friendly**: Reduces battery consumption vs polling

**Alternatives considered and rejected:**

❌ **Server-Sent Events (SSE)**: Unidirectional, less mobile-friendly, connection limits
❌ **Long polling**: Still creates connection overhead, complex timeout handling
❌ **gRPC streaming**: Overkill for this use case, client complexity

**Note on Gin Framework:** Gin does not have native WebSocket support built-in. The standard industry approach is to use Gin + gorilla/websocket together, which provides excellent integration and is the proven pattern for production Go WebSocket applications.

---

## Architecture Design

### High-Level Architecture

```
┌─────────────────┐
│  Mobile Client  │
│   (Flutter)     │
└────────┬────────┘
         │ WebSocket upgrade
         │ ws://server/api/v1/shops/:shop_id/messages/stream
         │
         ▼
┌─────────────────────────────────────────────────┐
│           Go Server (Gin Framework)             │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │   WebSocket Endpoint Handler             │  │
│  │   - Upgrade HTTP → WebSocket             │  │
│  │   - Authenticate user                    │  │
│  │   - Verify shop membership               │  │
│  └──────────┬───────────────────────────────┘  │
│             │                                   │
│             ▼                                   │
│  ┌──────────────────────────────────────────┐  │
│  │   Shop Message Hub (Connection Manager)  │  │
│  │                                          │  │
│  │   Map: ShopID → []WebSocket Clients     │  │
│  │                                          │  │
│  │   - Register new connections            │  │
│  │   - Unregister on disconnect            │  │
│  │   - Broadcast messages to shop members  │  │
│  └──────────┬───────────────────────────────┘  │
│             │                                   │
│             ▼                                   │
│  ┌──────────────────────────────────────────┐  │
│  │   Existing CreateShopMessage Handler     │  │
│  │   - Save message to DB                   │  │
│  │   - Notify hub to broadcast              │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
└─────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│   PostgreSQL    │
│  shop_messages  │
└─────────────────┘
```

### Component Responsibilities

#### 1. **WebSocket Handler** (New)
- Upgrade HTTP connection to WebSocket
- Authenticate user via JWT token
- Verify shop membership
- Register client with hub
- Handle ping/pong for connection health
- Clean up on disconnect

#### 2. **Shop Message Hub** (New)
- Maintain map of shop_id → active WebSocket clients
- Thread-safe client registration/unregistration
- Broadcast messages to all clients in a shop
- Handle graceful shutdown

#### 3. **WebSocket Client Wrapper** (New)
- Goroutine for reading (handles pings, close messages)
- Goroutine for writing (sends messages, handles backpressure)
- Buffered channel for outbound messages
- Connection health monitoring

#### 4. **Enhanced CreateShopMessage** (Modified)
- Existing DB insert logic (unchanged)
- **NEW**: Notify hub to broadcast to connected clients
- Fallback gracefully if broadcast fails

---

## Technical Implementation

### Core Components

#### 1. Hub Structure

```go
// File: api/websocket/shop_message_hub.go

package websocket

import (
    "sync"
    "miltechserver/.gen/miltech_ng/public/model"
)

// Hub maintains active WebSocket connections for all shops
type Hub struct {
    // Map: shop_id → set of clients
    shops map[string]map[*Client]bool

    // Register requests from clients
    register chan *Client

    // Unregister requests from clients
    unregister chan *Client

    // Broadcast messages to shop members
    broadcast chan *BroadcastMessage

    // Mutex for thread-safe map access
    mu sync.RWMutex
}

type BroadcastMessage struct {
    ShopID  string
    Message *model.ShopMessages
}

type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    shopID   string
    userID   string
    username string
}
```

#### 2. Client Connection Handling

```go
// File: api/websocket/client.go

// Read pump - handles incoming messages (pings, close)
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    // Read loop (discard client messages, we only broadcast server→client)
    for {
        if _, _, err := c.conn.ReadMessage(); err != nil {
            break
        }
    }
}

// Write pump - sends messages to client
func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

#### 3. Hub Run Loop

```go
// File: api/websocket/shop_message_hub.go

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            if _, ok := h.shops[client.shopID]; !ok {
                h.shops[client.shopID] = make(map[*Client]bool)
            }
            h.shops[client.shopID][client] = true
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            if clients, ok := h.shops[client.shopID]; ok {
                if _, ok := clients[client]; ok {
                    delete(clients, client)
                    close(client.send)
                    if len(clients) == 0 {
                        delete(h.shops, client.shopID)
                    }
                }
            }
            h.mu.Unlock()

        case msg := <-h.broadcast:
            h.mu.RLock()
            clients := h.shops[msg.ShopID]
            messageJSON, _ := json.Marshal(msg.Message)

            for client := range clients {
                select {
                case client.send <- messageJSON:
                default:
                    // Client send buffer full, disconnect
                    close(client.send)
                    delete(clients, client)
                }
            }
            h.mu.RUnlock()
        }
    }
}
```

#### 4. WebSocket Endpoint Handler

```go
// File: api/controller/shops_websocket_controller.go

package controller

import (
    "log/slog"
    "miltechserver/api/websocket"
    "miltechserver/bootstrap"
    "github.com/gin-gonic/gin"
    gorillaws "github.com/gorilla/websocket"
)

var upgrader = gorillaws.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // Production: validate origin against whitelist
        // Development: allow all
        return true
    },
}

func (controller *ShopsController) StreamShopMessages(c *gin.Context) {
    // 1. Authenticate user
    ctxUser, ok := c.Get("user")
    user, _ := ctxUser.(*bootstrap.User)
    if !ok {
        c.JSON(401, gin.H{"message": "unauthorized"})
        return
    }

    // 2. Get shop_id from URL
    shopID := c.Param("shop_id")
    if shopID == "" {
        c.JSON(400, gin.H{"message": "shop_id required"})
        return
    }

    // 3. Verify shop membership
    isMember, err := controller.ShopsService.IsUserShopMember(user, shopID)
    if err != nil || !isMember {
        c.JSON(403, gin.H{"message": "not a shop member"})
        return
    }

    // 4. Upgrade to WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        slog.Error("WebSocket upgrade failed", "error", err)
        return
    }

    // 5. Create client and register with hub
    client := websocket.NewClient(controller.Hub, conn, shopID, user.UserID, user.Username)
    controller.Hub.Register(client)

    // 6. Start client goroutines
    go client.WritePump()
    go client.ReadPump()

    slog.Info("WebSocket client connected", "shop_id", shopID, "user_id", user.UserID)
}
```

#### 5. Enhanced CreateShopMessage

```go
// File: api/service/shops_service_impl.go

func (service *ShopsServiceImpl) CreateShopMessage(user *bootstrap.User, message model.ShopMessages) (*model.ShopMessages, error) {
    // Existing validation and DB insert logic
    // ... (no changes to existing code)

    createdMessage, err := service.ShopsRepository.CreateShopMessage(user, message)
    if err != nil {
        return nil, err
    }

    // NEW: Broadcast to WebSocket clients (best-effort, non-blocking)
    if service.Hub != nil {
        service.Hub.BroadcastMessage(message.ShopID, createdMessage)
    }

    return createdMessage, nil
}
```

### Directory Structure

```
miltechserver/
├── api/
│   ├── controller/
│   │   ├── shops_controller.go              (existing)
│   │   └── shops_websocket_controller.go    (NEW)
│   ├── service/
│   │   ├── shops_service.go                 (modified - add Hub reference)
│   │   └── shops_service_impl.go            (modified - broadcast on create)
│   ├── websocket/                           (NEW PACKAGE)
│   │   ├── hub.go                          (Hub implementation)
│   │   ├── client.go                       (Client wrapper)
│   │   └── types.go                        (Shared types)
│   └── route/
│       └── shops_route.go                   (modified - add WebSocket route)
└── go.mod                                   (add github.com/gorilla/websocket)
```

---

## API Specification

### New WebSocket Endpoint

#### Connect to Shop Message Stream

**Endpoint**: `GET /api/v1/shops/:shop_id/messages/stream`
**Protocol**: WebSocket Upgrade
**Authentication**: Required (JWT token in query param or header)

**Request Headers**:
```
Connection: Upgrade
Upgrade: websocket
Sec-WebSocket-Version: 13
Sec-WebSocket-Key: <generated>
Authorization: Bearer <jwt_token>
```

**Connection Parameters**:
- `:shop_id` - UUID of the shop to receive messages from

**Authorization**:
- User must be authenticated
- User must be a member of the specified shop

**Response (Success - HTTP 101 Switching Protocols)**:
```
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: <generated>
```

**Response (Failure)**:
- `401 Unauthorized` - Invalid or missing JWT token
- `403 Forbidden` - User is not a member of the shop
- `400 Bad Request` - Missing shop_id parameter

### WebSocket Message Format

#### Server → Client Messages

When a new message is created in the shop, the server broadcasts:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "shop_id": "123e4567-e89b-12d3-a456-426614174000",
  "user_id": "789e0123-e45b-67c8-d901-234567890abc",
  "message": "Hello team!",
  "created_at": "2026-01-03T15:30:00Z",
  "updated_at": null,
  "is_edited": false
}
```

**Enhanced message with user context** (optional enhancement):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "shop_id": "123e4567-e89b-12d3-a456-426614174000",
  "user_id": "789e0123-e45b-67c8-d901-234567890abc",
  "username": "john_doe",
  "message": "Hello team!",
  "created_at": "2026-01-03T15:30:00Z",
  "updated_at": null,
  "is_edited": false
}
```

#### Client → Server Messages

Clients should NOT send messages over WebSocket (read-only stream). To send messages, clients continue using:

```
POST /api/v1/shops/messages
```

#### Ping/Pong (Connection Health)

**Server sends**: Ping frame every 54 seconds
**Client responds**: Pong frame
**Timeout**: Connection closed if no pong received within 60 seconds

### Client Connection Lifecycle

```
1. Client initiates WebSocket upgrade request
2. Server authenticates and validates shop membership
3. Connection established (HTTP 101)
4. Server registers client with hub
5. Client receives real-time messages as they're sent
6. Server sends periodic ping frames
7. Client responds with pong frames
8. On disconnect (network issue, app background):
   - Server detects missed pong
   - Server unregisters client
   - Client reconnects and polls API for missed messages
```

---

## Security Considerations

### Authentication & Authorization

#### WebSocket Upgrade Authentication

**Option 1: Query Parameter (Recommended for mobile clients)**
```
ws://server/api/v1/shops/:shop_id/messages/stream?token=<jwt_token>
```

**Option 2: Authorization Header**
```
Authorization: Bearer <jwt_token>
```

**Implementation**:
```go
func (controller *ShopsController) StreamShopMessages(c *gin.Context) {
    // Try header first
    token := c.GetHeader("Authorization")
    if token == "" {
        // Fallback to query param
        token = c.Query("token")
    }

    // Validate JWT and extract user
    user, err := controller.AuthService.ValidateToken(token)
    if err != nil {
        c.JSON(401, gin.H{"message": "unauthorized"})
        return
    }

    // Continue with shop membership check...
}
```

### Permission Verification

✅ **User authentication**: JWT token required
✅ **Shop membership**: Verified before WebSocket upgrade
✅ **Origin validation**: CheckOrigin configured for production
✅ **No privilege escalation**: Users only receive messages from shops they belong to

### Origin Validation

**Development**:
```go
CheckOrigin: func(r *http.Request) bool {
    return true // Allow all origins
}
```

**Production**:
```go
CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    allowedOrigins := []string{
        "https://yourdomain.com",
        "https://app.yourdomain.com",
    }
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

### Rate Limiting (Future Enhancement)

**Connection rate limiting** (per user):
- Max 5 WebSocket connections per user across all shops
- Max 10 connection attempts per minute

**Message rate limiting**:
- Existing REST API rate limits apply to `POST /shops/messages`

### Data Privacy

✅ **Broadcast isolation**: Messages only sent to clients connected to the same shop
✅ **No message history**: WebSocket only delivers NEW messages (no replay)
✅ **Secure transport**: WSS (WebSocket Secure) required in production

---

## Deployment Strategy

### Phase 1: Development & Testing

1. **Add dependency** (gorilla/websocket for WebSocket protocol support):
   ```bash
   go get github.com/gorilla/websocket@latest
   ```

   **Why gorilla/websocket?** Gin framework does not include native WebSocket support. The standard production approach is Gin + gorilla/websocket, which provides the protocol implementation while Gin handles routing and middleware.

2. **Implement core components**:
   - Create `api/websocket/` package
   - Implement Hub, Client, and types
   - Add WebSocket controller method
   - Register route in shops_route.go

3. **Local testing**:
   - Use WebSocket client tools (Postman, wscat)
   - Test authentication and authorization
   - Verify message broadcasting
   - Test reconnection handling

### Phase 2: Integration with Existing API

1. **Modify CreateShopMessage service**:
   - Add hub reference
   - Call broadcast after DB insert

2. **Add graceful degradation**:
   - WebSocket failures don't break message creation
   - Log errors for monitoring

3. **Update service initialization**:
   ```go
   // main.go or bootstrap
   hub := websocket.NewHub()
   go hub.Run() // Start hub in background goroutine

   shopsService := service.NewShopsServiceImpl(repository, hub)
   ```

### Phase 3: Production Deployment

1. **Environment configuration**:
   ```env
   WEBSOCKET_ENABLED=true
   WEBSOCKET_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
   WEBSOCKET_MAX_CONNECTIONS_PER_USER=5
   ```

2. **Load balancer configuration**:
   - Enable WebSocket support (sticky sessions if multi-instance)
   - Increase timeout values for persistent connections
   - Configure health checks

3. **Monitoring setup**:
   - Track active WebSocket connections
   - Monitor broadcast latency
   - Alert on connection failures

### Rollback Strategy

If issues arise:

1. **Disable WebSocket endpoint** (feature flag):
   ```go
   if !env.WebSocketEnabled {
       return // Skip WebSocket route registration
   }
   ```

2. **Clients automatically fall back to polling** (no code changes needed)

3. **Investigate issues without affecting core messaging**

---

## Testing Strategy

### Unit Tests

#### Hub Tests
```go
// api/websocket/hub_test.go

func TestHub_RegisterClient(t *testing.T) {
    hub := NewHub()
    client := &Client{shopID: "test-shop"}

    hub.register <- client
    // Verify client registered in hub.shops
}

func TestHub_BroadcastMessage(t *testing.T) {
    hub := NewHub()
    // Register multiple clients
    // Broadcast message
    // Verify all clients receive message
}
```

#### Client Tests
```go
// api/websocket/client_test.go

func TestClient_WritePump(t *testing.T) {
    // Test message sending
    // Test ping sending
    // Test graceful close
}

func TestClient_ReadPump(t *testing.T) {
    // Test pong handling
    // Test read deadline
    // Test disconnect handling
}
```

### Integration Tests

#### WebSocket Connection Test
```go
// api/controller/shops_websocket_test.go

func TestShopsController_StreamShopMessages_Success(t *testing.T) {
    // Setup: Create test shop, user, JWT
    // Connect WebSocket
    // Verify HTTP 101 response
    // Verify client registered
}

func TestShopsController_StreamShopMessages_Unauthorized(t *testing.T) {
    // Connect without JWT
    // Verify 401 response
}

func TestShopsController_StreamShopMessages_NotMember(t *testing.T) {
    // Connect with valid user who is not shop member
    // Verify 403 response
}
```

#### End-to-End Messaging Test
```go
func TestRealTimeMessaging_E2E(t *testing.T) {
    // 1. Connect WebSocket client for User A
    // 2. Connect WebSocket client for User B (same shop)
    // 3. User A sends message via POST /shops/messages
    // 4. Verify both WebSocket clients receive message
    // 5. Disconnect User B
    // 6. User A sends another message
    // 7. Verify only User A receives (User B disconnected)
}
```

### Load Testing

#### Connection Scaling
```bash
# Test 100 concurrent connections
wscat -c "ws://localhost:8080/api/v1/shops/test-shop/messages/stream?token=<jwt>" &
# Repeat 100 times
```

#### Message Broadcasting
```go
// Benchmark test
func BenchmarkHub_Broadcast_100Clients(b *testing.B) {
    hub := NewHub()
    // Register 100 clients
    // Broadcast b.N messages
    // Measure throughput
}
```

**Target metrics**:
- Support 1000+ concurrent connections per server instance
- Message delivery latency < 50ms
- Memory usage < 10MB per 100 connections

### Manual Testing Checklist

- [ ] WebSocket connection succeeds with valid JWT
- [ ] WebSocket connection fails with invalid JWT
- [ ] WebSocket connection fails for non-member
- [ ] New messages broadcast to all connected clients
- [ ] Edited messages broadcast to all connected clients
- [ ] Deleted messages broadcast to all connected clients (future)
- [ ] Client receives ping frames
- [ ] Client disconnect unregisters from hub
- [ ] Reconnection works after disconnect
- [ ] Multiple shops operate independently (no cross-shop leaks)
- [ ] Origin validation works in production mode

---

## Performance Considerations

### Memory Usage

**Per-client overhead**:
- WebSocket connection: ~4KB
- Send buffer (256 messages): ~64KB
- Client struct: ~1KB
- **Total per client**: ~70KB

**Scaling estimates**:
- 100 clients = ~7MB
- 1,000 clients = ~70MB
- 10,000 clients = ~700MB

### CPU Usage

**Hub operations**:
- Client registration: O(1)
- Client unregistration: O(1)
- Message broadcast: O(n) where n = clients in shop

**Optimization**: Shops are independent, so broadcasting to one shop doesn't affect others.

### Network Bandwidth

**Ping/Pong overhead**:
- Ping frame every 54 seconds
- Pong response
- **Total**: ~2 bytes/minute per connection (negligible)

**Message broadcasting**:
- Average message size: ~500 bytes (JSON)
- Broadcast to n clients: 500 * n bytes per message

### Database Load Reduction

**Before (polling every 5 seconds)**:
- 100 active users = 1,200 queries/minute
- Constant DB load even when no messages

**After (WebSocket)**:
- 100 active users = 0 queries (unless sending)
- DB queries only on message creation

**Estimated reduction**: 90-95% fewer read queries

### Scalability Limits (Single Server)

**Expected capacity** (based on research):
- **Concurrent connections**: 5,000-10,000
- **Active shops**: 500-1,000 simultaneously
- **Messages/second**: 500-1,000

**Bottlenecks**:
1. CPU for JSON marshaling (broadcast)
2. Memory for client buffers
3. File descriptors (OS limit)

### Horizontal Scaling (Future)

If single-server limits are reached:

**Option 1: Load balancer with sticky sessions**
- Simple, no code changes
- Clients pinned to server instance
- Shop members may be on different servers (no real-time updates)

**Option 2: Redis Pub/Sub (recommended for large scale)**
- Add Redis dependency
- Hub publishes messages to Redis channel
- All server instances subscribe and forward to local clients
- Enables true multi-server real-time messaging

---

## Mobile Client Integration

### Flutter WebSocket Client (Example)

```dart
import 'package:web_socket_channel/web_socket_channel.dart';

class ShopMessageService {
  WebSocketChannel? _channel;
  final String shopId;
  final String jwtToken;

  ShopMessageService({required this.shopId, required this.jwtToken});

  void connect() {
    final uri = Uri.parse(
      'ws://your-server.com/api/v1/shops/$shopId/messages/stream?token=$jwtToken'
    );

    _channel = WebSocketChannel.connect(uri);

    _channel!.stream.listen(
      (message) {
        // Parse JSON and update UI
        final msg = ShopMessage.fromJson(jsonDecode(message));
        _onNewMessage(msg);
      },
      onError: (error) {
        print('WebSocket error: $error');
        _reconnect();
      },
      onDone: () {
        print('WebSocket closed');
        _reconnect();
      },
    );
  }

  void _reconnect() {
    Future.delayed(Duration(seconds: 2), () {
      connect();
      _fetchMissedMessages(); // Poll API for missed messages
    });
  }

  Future<void> _fetchMissedMessages() async {
    // Call GET /shops/:shop_id/messages?since=<last_timestamp>
  }

  void disconnect() {
    _channel?.sink.close();
  }
}
```

### Client Best Practices

1. **Exponential backoff on reconnect**:
   - First retry: 1 second
   - Second retry: 2 seconds
   - Third retry: 4 seconds
   - Max: 30 seconds

2. **Heartbeat monitoring**:
   - Detect when server stops sending pings
   - Proactively reconnect

3. **Background handling**:
   - Disconnect WebSocket when app backgrounds
   - Reconnect and poll on app foreground

4. **Offline mode**:
   - Queue messages for send when online
   - Show cached messages immediately

---

## Migration Path

### Phase 1: Backend Implementation (Week 1)

**Step 1**: Add WebSocket infrastructure
- Install gorilla/websocket dependency
- Create websocket package (hub, client, types)
- Add unit tests

**Step 2**: Add WebSocket endpoint
- Implement StreamShopMessages controller
- Add route registration
- Add integration tests

**Step 3**: Integrate with message creation
- Modify CreateShopMessage to broadcast
- Add graceful error handling
- Test end-to-end flow

### Phase 2: Client Integration (Week 2)

**Step 4**: Mobile app WebSocket client
- Implement connection manager
- Add reconnection logic
- Add fallback polling

**Step 5**: UI updates
- Subscribe to WebSocket messages
- Update message list in real-time
- Show connection status indicator

### Phase 3: Production Deployment (Week 3)

**Step 6**: Staging deployment
- Deploy to staging environment
- Load testing with realistic traffic
- Monitor performance metrics

**Step 7**: Production rollout
- Deploy to production
- Monitor error rates and latency
- Gradual rollout to users (feature flag)

**Step 8**: Deprecate polling (Optional - Week 4+)
- After 2-4 weeks of stable WebSocket operation
- Keep polling as fallback indefinitely

---

## Monitoring & Observability

### Metrics to Track

**Connection metrics**:
- `websocket_connections_total` - Active WebSocket connections
- `websocket_connections_by_shop` - Connections per shop
- `websocket_connection_duration_seconds` - Connection lifetime distribution

**Message metrics**:
- `websocket_messages_broadcast_total` - Messages broadcast
- `websocket_broadcast_latency_ms` - Time to broadcast to all clients
- `websocket_broadcast_errors_total` - Failed broadcasts

**Error metrics**:
- `websocket_auth_failures_total` - Authentication failures
- `websocket_upgrade_failures_total` - Connection upgrade failures
- `websocket_disconnect_reasons` - Categorized disconnect causes

### Logging Strategy

**Connection events**:
```go
slog.Info("WebSocket client connected",
    "shop_id", shopID,
    "user_id", userID,
    "username", username,
)

slog.Info("WebSocket client disconnected",
    "shop_id", shopID,
    "user_id", userID,
    "duration_seconds", time.Since(connectedAt).Seconds(),
)
```

**Broadcast events**:
```go
slog.Debug("Broadcasting message",
    "shop_id", shopID,
    "message_id", messageID,
    "recipient_count", len(clients),
)
```

**Error events**:
```go
slog.Error("WebSocket upgrade failed",
    "error", err,
    "shop_id", shopID,
    "user_id", userID,
)
```

### Health Checks

**WebSocket endpoint health**:
```
GET /api/v1/health/websocket
```

Response:
```json
{
  "status": "healthy",
  "active_connections": 1234,
  "active_shops": 56,
  "uptime_seconds": 86400
}
```

---

## Future Enhancements

### Phase 2 Features (Post-MVP)

1. **Typing indicators**:
   - Client → Server: "User is typing"
   - Server → Clients: Broadcast typing status

2. **Read receipts**:
   - Client → Server: "Message marked as read"
   - Server → Clients: Broadcast read status

3. **Message reactions** (emoji):
   - Add reaction via WebSocket or REST
   - Broadcast reaction to all clients

4. **Presence indicators**:
   - Show who's online in the shop
   - Broadcast join/leave events

### Horizontal Scaling (When Needed)

**Redis Pub/Sub Integration**:
```go
// Hub publishes to Redis
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
pubsub := rdb.Subscribe(ctx, fmt.Sprintf("shop:%s:messages", shopID))

// All server instances subscribe and forward to local clients
go func() {
    for msg := range pubsub.Channel() {
        h.broadcastToLocalClients(shopID, msg.Payload)
    }
}()
```

**Benefits**:
- Support 10,000+ concurrent connections
- Distribute load across multiple servers
- High availability (server failure doesn't lose all connections)

### Advanced Features

1. **Message edit/delete notifications**:
   - Broadcast edit events over WebSocket
   - Clients update UI in real-time

2. **File upload progress**:
   - Stream upload progress over WebSocket
   - Show live progress bar

3. **Voice message streaming**:
   - Stream audio chunks as they're uploaded
   - Real-time playback for recipients

---

## References

### Research Sources

This design is informed by industry best practices from 2025-2026:

**Gin + WebSocket Integration**:
- [Go Websockets (Gin-gonic + Gorilla) - Silversmith](https://arlimus.github.io/articles/gin.and.gorilla/)
- [Building WebSocket for Notifications with GoLang and Gin - Medium](https://medium.com/@abhishekranjandev/building-a-production-grade-websocket-for-notifications-with-golang-and-gin-a-detailed-guide-5b676dcfbd5a)
- [Real-time Messaging with Go: Gin, Websockets, and RabbitMQ - Medium](https://medium.com/@tanngontn/golang-gin-framework-with-normal-websocket-and-websocket-with-producer-is-rabbitmq-guide-93cad7d290f7)
- [Build a Simple Websocket Server in Go with Gin](https://lwebapp.com/en/post/go-websocket-simple-server)

**WebSocket Best Practices**:
- [Go WebSocket: A Comprehensive Guide to Real-Time Communication in Go (2025) - VideoSDK](https://www.videosdk.live/developer-hub/websocket/go-websocket)
- [Real-Time Communication with Gorilla WebSocket in Go Applications | Leapcell](https://leapcell.io/blog/real-time-communication-with-gorilla-websocket-in-go-applications)
- [websocket package - github.com/gorilla/websocket - Go Packages](https://pkg.go.dev/github.com/gorilla/websocket)
- [GitHub - gorilla/websocket](https://github.com/gorilla/websocket)

**Scalability & Performance**:
- [Building a Scalable Go WebSocket Service for Thousands of Concurrent Connections | Leapcell](https://leapcell.io/blog/building-a-scalable-go-websocket-service-for-thousands-of-concurrent-connections)
- [Optimizing Websocket Broadcasting in Go - DEV Community](https://dev.to/aaravjoshi/optimizing-websocket-broadcasting-in-go-strategies-for-high-performance-real-time-applications-1kai)
- [Go WebSocket Scaling: How to Minimize Your Footprint | Druva](https://www.druva.com/blog/websockets--scale-at-fractional-footprint-in-go)
- [Scaling to a Million WebSockets with Go - Bomberbot](https://www.bomberbot.com/golang/scaling-to-a-million-websockets-with-go/)

### Technical Standards

- **WebSocket Protocol**: RFC 6455
- **JSON Message Format**: RFC 8259
- **JWT Authentication**: RFC 7519

### Library Documentation

- **gorilla/websocket**: https://pkg.go.dev/github.com/gorilla/websocket
- **Gin Framework**: https://gin-gonic.com/docs/
- **Go Jet ORM**: https://github.com/go-jet/jet

---

## Appendix: Configuration Reference

### Environment Variables

```env
# WebSocket Configuration
WEBSOCKET_ENABLED=true
WEBSOCKET_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com
WEBSOCKET_READ_BUFFER_SIZE=1024
WEBSOCKET_WRITE_BUFFER_SIZE=1024
WEBSOCKET_PING_PERIOD_SECONDS=54
WEBSOCKET_PONG_WAIT_SECONDS=60
WEBSOCKET_WRITE_WAIT_SECONDS=10
WEBSOCKET_MAX_CONNECTIONS_PER_USER=5
```

### Timing Constants

```go
const (
    // Time allowed to write a message to the peer
    writeWait = 10 * time.Second

    // Time allowed to read the next pong message from the peer
    pongWait = 60 * time.Second

    // Send pings to peer with this period (must be less than pongWait)
    pingPeriod = (pongWait * 9) / 10 // 54 seconds

    // Maximum message size allowed from peer
    maxMessageSize = 512 * 1024 // 512KB

    // Send buffer size (number of messages)
    sendBufferSize = 256
)
```

---

## Questions & Answers

### Q: Why not use Server-Sent Events (SSE)?

**A**: While SSE is simpler, WebSocket is preferred because:
- SSE is unidirectional (server→client only)
- SSE has browser connection limits (6 per domain)
- WebSocket is more efficient for mobile apps
- WebSocket allows future bi-directional features (typing indicators, etc.)

### Q: What happens if the WebSocket connection drops?

**A**: The client reconnects and polls the REST API (`GET /shops/:shop_id/messages`) to fetch missed messages. This is the "simple reconnect" strategy.

### Q: Can we support multiple server instances?

**A**: Yes, but requires Redis Pub/Sub for message distribution. The initial implementation supports single-server deployment. Horizontal scaling is a Phase 2 enhancement.

### Q: How do we prevent unauthorized users from connecting?

**A**: Three layers of security:
1. JWT authentication (validates user identity)
2. Shop membership check (validates user has access to shop)
3. Origin validation (prevents CSRF in production)

### Q: What's the latency for message delivery?

**A**: Expected latency: 10-50ms from message creation to client receipt (same server). This is 100x faster than typical polling intervals (5-10 seconds).

---

## Approval & Sign-off

**Design Status**: ✅ Ready for implementation
**Estimated Effort**: 3-4 days development + 1-2 days testing
**Dependencies**: gorilla/websocket library
**Risk Level**: Low (graceful fallback to existing polling)

**Next Steps**:
1. Review and approve design document
2. Create implementation tasks in project management system
3. Begin Phase 1 development (backend WebSocket infrastructure)
4. Coordinate with mobile team for client integration

---

**Document Version**: 1.0
**Last Updated**: 2026-01-03
**Author**: swisscheese
