# Mobile WebSocket Client Design Document

## Executive Summary

This document provides specifications for implementing WebSocket real-time messaging in the mobile application. It covers connection management, authentication, message handling, error recovery, and client behavior patterns.

**Target Audience**: Mobile development team (Flutter/iOS/Android)
**Server Version**: v1.0 (2026-01-03)
**Status**: Ready for implementation
**Created**: 2026-01-03

---

## Table of Contents

1. [Overview](#overview)
2. [Connection Specifications](#connection-specifications)
3. [Authentication](#authentication)
4. [Message Format](#message-format)
5. [Connection Lifecycle](#connection-lifecycle)
6. [Error Handling](#error-handling)
7. [Background Behavior](#background-behavior)
8. [Network Transition Handling](#network-transition-handling)
9. [Reconnection Strategy](#reconnection-strategy)
10. [Data Synchronization](#data-synchronization)
11. [Performance Requirements](#performance-requirements)
12. [Security Considerations](#security-considerations)

---

## Overview

### Purpose

The WebSocket feature enables real-time message delivery for shop messaging, eliminating the need for polling. Messages are delivered instantly (10-50ms latency) when sent by any shop member.

### Key Benefits for Mobile Users

- **Instant message delivery**: No waiting for polling intervals
- **Battery efficiency**: 90% reduction in network requests compared to polling
- **Bandwidth savings**: Only receive messages when they exist, no empty poll responses
- **Better UX**: Real-time chat experience similar to modern messaging apps

### Architecture Pattern

The mobile client maintains a **read-only WebSocket stream**:
- **Receive**: Messages broadcast from server in real-time
- **Send**: Messages sent via existing REST API (`POST /shops/messages`)
- **Fallback**: Automatically fall back to polling if WebSocket unavailable

---

## Connection Specifications

### WebSocket Endpoint

```
wss://your-server.com/api/v1/auth/shops/{shop_id}/messages/stream
```

**Protocol**: WebSocket (WSS in production, WS in development)
**Path Parameters**:
- `{shop_id}`: UUID of the shop to receive messages from

**Query Parameters**:
- `token`: (Optional) JWT authentication token if not using Authorization header

### Connection Requirements

- **Authentication**: Required (JWT token)
- **Authorization**: User must be a member of the specified shop
- **Protocol Version**: WebSocket Protocol Version 13 (RFC 6455)
- **Subprotocols**: None required
- **Extensions**: None required

### Connection Limits

- **Maximum connections per user**: 5 concurrent connections across all devices
- **Maximum connection duration**: Unlimited (server sends periodic pings)
- **Idle timeout**: 60 seconds (if client stops responding to pings)

---

## Authentication

### Authentication Methods

The server supports **two authentication methods**. Choose the one that works best for your WebSocket library:

#### Method 1: Query Parameter (Recommended for Mobile)

Include JWT token as a query parameter:

```
wss://server.com/api/v1/auth/shops/{shop_id}/messages/stream?token={jwt_token}
```

**Advantages**:
- Works with all WebSocket libraries
- No header manipulation required
- Simpler implementation

**Security Note**: Token is visible in connection URL but connection is encrypted (WSS)

#### Method 2: Authorization Header (Alternative)

Include JWT token in the WebSocket upgrade request headers:

```
Authorization: Bearer {jwt_token}
```

**Advantages**:
- Follows REST API authentication pattern
- Token not in URL logs

**Note**: Some WebSocket libraries may not support custom headers during upgrade

### Authentication Flow

1. **Obtain JWT token** from existing authentication system
2. **Initiate WebSocket connection** with token (query param or header)
3. **Server validates** token and shop membership
4. **Connection succeeds** (HTTP 101 Switching Protocols) or **fails** with error

### Authentication Errors

| HTTP Status | Reason | Client Action |
|-------------|--------|---------------|
| 401 Unauthorized | Invalid or expired JWT token | Refresh token and retry connection |
| 403 Forbidden | User not a member of shop | Display error, prevent reconnection |
| 400 Bad Request | Missing shop_id parameter | Fix request and retry |
| 503 Service Unavailable | WebSocket service disabled | Fall back to polling mode |

---

## Message Format

### Incoming Messages (Server → Client)

All messages from the server are JSON-formatted with the following structure:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "shop_id": "123e4567-e89b-12d3-a456-426614174000",
  "user_id": "789e0123-e45b-67c8-d901-234567890abc",
  "username": "john_doe",
  "message": "Hello team! The vehicle inspection is complete.",
  "created_at": "2026-01-03T15:30:00Z",
  "updated_at": null,
  "is_edited": false
}
```

### Field Specifications

| Field | Type | Description | Nullable |
|-------|------|-------------|----------|
| `id` | String (UUID) | Unique message identifier | No |
| `shop_id` | String (UUID) | Shop this message belongs to | No |
| `user_id` | String (UUID) | User who sent the message | No |
| `username` | String | Display name of sender | No |
| `message` | String | Message content (max 10,000 chars) | No |
| `created_at` | String (ISO 8601) | Message creation timestamp | No |
| `updated_at` | String (ISO 8601) | Last edit timestamp | Yes |
| `is_edited` | Boolean | Whether message has been edited | No |

### Message Timestamps

- **Format**: ISO 8601 with timezone (e.g., `2026-01-03T15:30:00Z`)
- **Timezone**: UTC (denoted by `Z` suffix)
- **Client Handling**: Convert to local timezone for display

### Message Ordering

- Messages are **not guaranteed** to arrive in chronological order via WebSocket
- Client **must sort** messages by `created_at` timestamp before displaying

### Special Message Types

**Initial Connection**:
- Server **does NOT** send message history on connect
- Client must fetch history via REST API if needed

**Ping/Pong Frames**:
- Server sends ping every 54 seconds
- Client must respond with pong (handled automatically by most WebSocket libraries)
- **Not visible** to application layer

---

## Connection Lifecycle

### 1. Pre-Connection Phase

**Client Actions**:
1. User navigates to shop message screen
2. Verify user is authenticated (valid JWT token)
3. Verify user is member of the shop (check local cache or API)
4. Check if WebSocket is supported and enabled

**Preconditions**:
- Valid JWT token available
- User confirmed as shop member
- Network connectivity available

### 2. Connection Establishment

**Client Actions**:
1. Construct WebSocket URL with shop_id and token
2. Initiate WebSocket connection
3. Wait for connection event or timeout (5 seconds)

**Success Indicators**:
- WebSocket `onOpen` event fires
- Connection state changes to `OPEN`

**Failure Handling**:
- If timeout (5s), retry with exponential backoff
- If authentication error (401), refresh token and retry
- If authorization error (403), display error and stop

### 3. Active Connection Phase

**Client Responsibilities**:
- Listen for incoming messages
- Parse JSON message payloads
- Update UI with new messages
- Respond to ping frames (automatic in most libraries)
- Monitor connection health

**Server Behavior**:
- Broadcasts new messages as they're created
- Sends ping frames every 54 seconds
- Expects pong response within 60 seconds

### 4. Message Reception

**Message Handling Flow**:
1. Receive WebSocket message event
2. Parse JSON payload
3. Validate message structure and shop_id
4. Check for duplicates (by message `id`)
5. Insert into local message list (sorted by timestamp)
6. Update UI to display message
7. Mark as delivered (optional)

**Duplicate Detection**:
- Server may send same message multiple times on reconnection
- Client **must deduplicate** by message `id`
- Use Set/Map data structure for O(1) duplicate checking

### 5. Connection Termination

**Normal Disconnection** (Client-Initiated):
1. User navigates away from message screen
2. App moves to background
3. Client closes WebSocket connection cleanly
4. Server detects close and unregisters client

**Abnormal Disconnection** (Network/Server):
1. Network loss detected
2. Server timeout (no pong response)
3. WebSocket `onClose` or `onError` event fires
4. Client initiates reconnection strategy

### 6. Post-Disconnection

**Client Actions**:
- Clear connection state
- Preserve received messages in local storage
- Prepare for reconnection if needed
- Fall back to polling if reconnection fails repeatedly

---

## Error Handling

### Connection Errors

| Error Type | When It Occurs | Client Action |
|------------|----------------|---------------|
| Network unavailable | No internet connection | Display offline indicator, queue for retry when online |
| DNS resolution failed | Server domain unreachable | Retry with exponential backoff (max 5 attempts) |
| Connection timeout | Server not responding | Retry with exponential backoff |
| Connection refused | Server port closed | Fall back to polling mode |
| SSL/TLS error | Certificate validation failed | Display security error, prevent connection |

### Authentication/Authorization Errors

| Error | HTTP Status | Client Action |
|-------|-------------|---------------|
| Token expired | 401 | Refresh JWT token silently, retry connection |
| Token invalid | 401 | Force user re-login |
| Not shop member | 403 | Display error message, prevent further attempts |
| User banned/removed | 403 | Redirect to shop list, show removal notification |

### Runtime Errors

| Error Type | Cause | Client Action |
|------------|-------|---------------|
| Message parse error | Invalid JSON received | Log error, ignore message, continue connection |
| Unknown message type | Server sent unrecognized format | Log warning, ignore message |
| Send buffer full | Client sending too fast | This should never happen (read-only stream) |
| Ping timeout | No ping received for 90s | Assume connection dead, initiate reconnect |

### Error Recovery Strategy

**Transient Errors** (network, timeout):
- Retry with exponential backoff
- Max 5 attempts: 1s, 2s, 4s, 8s, 16s
- After 5 failures, fall back to polling

**Permanent Errors** (403 Forbidden):
- Do not retry
- Display user-friendly error message
- Disable WebSocket for this shop

**Critical Errors** (security, parsing):
- Log error details for debugging
- Close connection immediately
- Fall back to polling mode

---

## Background Behavior

### When App Backgrounds

**Client Actions**:
1. Detect app moving to background (lifecycle event)
2. Immediately close WebSocket connection
3. Preserve current message state in local storage
4. Store timestamp of last received message

**Rationale**:
- Conserves battery (no active network connection)
- Prevents iOS/Android from killing connection
- Reduces server resource usage
- Aligns with mobile platform best practices

### When App Foregrounds

**Client Actions**:
1. Detect app returning to foreground (lifecycle event)
2. **Do NOT immediately reconnect WebSocket**
3. Call REST API to fetch missed messages: `GET /shops/{shop_id}/messages/paginated?since={last_timestamp}`
4. Update local message list with missed messages
5. **After** fetching missed messages, reconnect WebSocket
6. Resume real-time message stream

**Rationale**:
- Ensures no messages are missed during background period
- Prevents race condition (WebSocket connects before fetching missed messages)
- Provides seamless user experience

### Background Time Estimation

| Background Duration | Sync Strategy |
|---------------------|---------------|
| < 5 minutes | Fetch paginated messages (likely 0-5 new messages) |
| 5-60 minutes | Fetch paginated messages (likely 5-50 new messages) |
| > 60 minutes | Fetch last 100 messages to ensure full context |

---

## Network Transition Handling

### Network Change Detection

**Monitor for**:
- WiFi → Cellular transition
- Cellular → WiFi transition
- Network loss → Network restored
- IP address change
- Airplane mode toggle

**Platform APIs**:
- Flutter: `connectivity_plus` package

### Transition Response

**When Network Changes**:

1. **Detect Change**:
   - Network monitoring callback fires
   - Connection state changes

2. **Immediate Action**:
   - Close existing WebSocket connection (if any)
   - Mark connection as "transitioning"

3. **Wait for Stability** (2 seconds):
   - Allow network to fully establish
   - Prevents rapid reconnect attempts during unstable transition

4. **Reconnect**:
   - Initiate new WebSocket connection
   - Fetch missed messages (from last received timestamp to now)
   - Resume normal operation

**Special Case: Network Loss**:
- Connection will close automatically
- Trigger reconnection strategy (exponential backoff)
- Display offline indicator if desired
- Fall back to polling after 5 failed attempts

---

## Reconnection Strategy

### Exponential Backoff Algorithm

When connection fails or disconnects unexpectedly:

**Attempt Schedule**:
1. **Attempt 1**: Immediate retry (0 seconds)
2. **Attempt 2**: After 1 second
3. **Attempt 3**: After 2 seconds (cumulative: 3s)
4. **Attempt 4**: After 4 seconds (cumulative: 7s)
5. **Attempt 5**: After 8 seconds (cumulative: 15s)

**After 5 Failed Attempts**:
- Abandon WebSocket reconnection
- Fall back to **polling mode** (poll every 30 seconds)
- Stop retry attempts to conserve battery

### Reconnection Triggers

**Automatic Reconnection**:
- Connection lost unexpectedly
- Network restored after loss
- Ping timeout (no ping received for 90s)

**Manual Reconnection**:
- User sends new message via REST API
- User pulls-to-refresh message list
- App returns to foreground

**Do NOT Reconnect**:
- 403 Forbidden (not shop member)
- WebSocket service disabled (503)
- User explicitly disabled real-time messaging in settings

### Connection State Management

**States**:
- `DISCONNECTED`: No connection, not attempting to reconnect
- `CONNECTING`: Attempting to establish connection
- `CONNECTED`: Active WebSocket connection established
- `RECONNECTING`: Connection lost, attempting to reconnect (with attempt count)
- `FAILED`: All reconnection attempts exhausted, using polling fallback

**State Transitions**:
```
DISCONNECTED → CONNECTING → CONNECTED
                    ↓
                RECONNECTING → CONNECTED
                    ↓
                  FAILED → (Polling Mode)
```

---

## Data Synchronization

### Initial Load Strategy

**When User Opens Message Screen**:

1. **Fetch Message History** (REST API):
   ```
   GET /api/v1/auth/shops/{shop_id}/messages/paginated?page=1&limit=50
   ```
   - Load most recent 50 messages
   - Display in UI immediately
   - Store `last_message_timestamp` from newest message

2. **Connect WebSocket**:
   - Only after message history loaded
   - Begin receiving real-time messages
   - Append to existing message list

3. **User Scrolls Up** (pagination):
   - Fetch older messages via REST API
   - Append to top of message list
   - Continue WebSocket connection for new messages

### Missed Message Recovery

**Scenario**: Connection was lost, now reconnected

**Recovery Process**:

1. **Identify Gap**:
   - Last received message timestamp: `T1`
   - Current time: `T2`
   - Gap duration: `T2 - T1`

2. **Fetch Missed Messages** (REST API):
   ```
   GET /api/v1/auth/shops/{shop_id}/messages/paginated?since={T1}&limit=100
   ```
   - Fetch all messages created after `T1`
   - Merge with existing messages (deduplicate by `id`)
   - Sort by timestamp

3. **Resume Real-Time Stream**:
   - WebSocket now connected
   - New messages append to list
   - User sees seamless history

### Duplicate Prevention

**Why Duplicates Occur**:
- Message received via WebSocket
- Connection drops immediately after
- Client reconnects and fetches missed messages
- Same message delivered via REST API

**Prevention Strategy**:
1. Maintain Set of received message IDs
2. Before adding message to UI, check if `id` exists in Set
3. If duplicate, ignore
4. If new, add to Set and display

**Performance Optimization**:
- Use hash-based Set (O(1) lookup)
- Prune Set after 1000 messages to prevent memory growth
- Persist Set to local storage for across-session deduplication

---

## Performance Requirements

### Response Time

| Action | Target | Maximum |
|--------|--------|---------|
| Connection establishment | < 500ms | 2s |
| Message display (after receipt) | < 50ms | 200ms |
| Reconnection attempt | < 1s | 3s |
| Fallback to polling | < 5s | 10s |

### Resource Usage

| Resource | Target | Maximum |
|----------|--------|---------|
| Memory per connection | < 1MB | 5MB |
| CPU usage (idle) | < 1% | 5% |
| Battery drain (per hour) | < 1% | 3% |
| Network bandwidth (idle) | < 500 bytes/min | 2KB/min |

### Scalability

| Metric | Requirement |
|--------|-------------|
| Maximum simultaneous shop connections | 5 shops |
| Maximum messages in memory | 1,000 messages per shop |
| Message list scroll performance | 60 FPS with 1,000 messages |
| Message insertion latency | < 16ms (60 FPS) |

---

## Security Considerations

### Transport Security

**Requirements**:
- **Production**: MUST use WSS (WebSocket Secure) with TLS 1.2+
- **Development**: MAY use WS (unencrypted) for localhost testing
- **Certificate Validation**: MUST validate server SSL certificate
- **No Certificate Pinning**: Not required (standard CA validation sufficient)

### Authentication Security

**Token Handling**:
- **Never log** JWT tokens in plaintext
- **Never persist** tokens in unencrypted storage
- **Refresh tokens** proactively before expiration
- **Clear tokens** on logout/session end

**Token Exposure**:
- Query parameter method exposes token in URL
- Acceptable because connection is encrypted (WSS)
- Tokens are short-lived (typically 1 hour expiration)
- Connection URLs not persisted in server logs

### Message Security

**Content Validation**:
- Verify `shop_id` matches expected shop (prevent message leakage)
- Sanitize message content before display (prevent XSS if rendering HTML)
- Validate message size (reject > 10KB payloads)


### Network Security

**Connection Hijacking Prevention**:
- WSS encryption prevents man-in-the-middle attacks
- Token rotation prevents replay attacks
- Short connection lifetimes limit hijacking window

**Denial of Service Prevention**:
- Rate limit connection attempts (max 10/minute)
- Exponential backoff prevents server flooding
- Fallback to polling prevents infinite retry loops

---

## Implementation Checklist

### Phase 1: Basic WebSocket Connection

- [ ] Implement WebSocket connection with authentication (query param method)
- [ ] Handle connection success/failure states
- [ ] Parse incoming JSON messages
- [ ] Display messages in UI (basic)
- [ ] Implement connection close on app background

### Phase 2: Error Handling & Reconnection

- [ ] Implement exponential backoff reconnection strategy
- [ ] Add network change detection
- [ ] Implement missed message recovery on reconnect
- [ ] Add duplicate message detection
- [ ] Implement polling fallback after 5 failed attempts

### Phase 3: Performance & Polish

- [ ] Optimize message list rendering (virtual scrolling)
- [ ] Add message deduplication Set with pruning
- [ ] Implement local message persistence
- [ ] Optimize memory usage (limit in-memory messages)
- [ ] Add error logging for debugging

### Phase 4: Production Readiness

- [ ] Switch to WSS for production environment
- [ ] Implement certificate validation
- [ ] Add security headers validation
- [ ] Performance testing with 1000+ messages
- [ ] Battery usage profiling

---

## Frequently Asked Questions

### Q: Why not send messages via WebSocket?

**A**: The WebSocket is designed as a **read-only stream** for architectural simplicity:
- Simpler error handling (no acknowledgments needed)
- Existing REST API already handles message creation, validation, and persistence
- Reduces WebSocket server complexity
- Aligns with proven patterns (Discord, Slack use similar architecture)

### Q: What happens if WebSocket and REST API messages arrive out of order?

**A**: Client must **always sort by timestamp**:
- WebSocket delivers message at `T+0ms`
- REST API returns same message at `T+500ms` (polling)
- Client deduplicates by `id` and sorts by `created_at`
- User sees messages in correct chronological order

### Q: How do I know if a message was delivered?

**A**:
- WebSocket delivery is **best-effort**
- No delivery confirmations or read receipts in MVP
- If message appears via WebSocket OR polling, it was delivered

### Q: Can I connect to multiple shops simultaneously?

**A**: Yes, but with limits:
- Maximum 5 concurrent connections per user
- Each shop requires separate WebSocket connection
- Recommended: Only connect to currently-visible shop
- Close connections when user navigates away

### Q: What if the server doesn't support WebSocket?

**A**: Graceful degradation:
- Server returns 503 Service Unavailable on connection attempt
- Client immediately falls back to polling mode
- No user-visible error (transparent fallback)
- Polling continues until WebSocket becomes available

### Q: How do I handle message edits/deletes?

**A**: Not supported in MVP:
- Message edit/delete notifications require Phase 2 server enhancement
- For now, client must poll or refresh to see edits/deletes
- Future enhancement: WebSocket events for edits/deletes

---

## Support & Contact

**Questions**: Contact backend team for server-side clarifications
**Issues**: Report client-side issues to mobile team lead
**Server Endpoint Documentation**: See `DESIGN_SHOP_MESSAGES_WEBSOCKET.md` for server architecture

---

**Document Version**: 1.0
**Last Updated**: 2026-01-03
**Author**: Backend Team
**Next Review**: After mobile client implementation
