---
domain: Designs
status: Draft
entry_points:
  - internal/handler/messaging.go
  - internal/service/messaging.go
  - internal/repository/messaging.go
dependencies:
  - .aidoc/designs/database-schema.md
  - .aidoc/designs/connection-handshake.md
  - .aidoc/architecture/guidelines.md
---

# In-App Messaging — API Design

Text messaging between connected (matched) partners. Users who have accepted a connection can exchange messages within their match conversation.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Database Schema](database-schema.md) | `messages`, `matches` tables |
| [Connection Handshake](connection-handshake.md) | Accept flow creates the match + re-associates pre-accept messages |
| [In-App Messaging (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/in-app-messaging.md) | Frontend design for conversations hub and chat |
| [Architecture Guidelines](../architecture/guidelines.md) | Layer separation, i18n, error handling |
| [API Contract](api-contract.md) | Shared endpoint conventions |

## Why This Design Exists

With connections in place, users need a way to communicate. This design adds match-scoped messaging — listing conversations, reading message history, and sending new messages — all on top of the existing `messages` table.

## Data Model

### Existing Tables (No Schema Changes Required)

The `messages` table already supports match messaging via the `match_id` FK:

```sql
-- From migration 00007
messages (
  id UUID PRIMARY KEY,
  match_id UUID REFERENCES matches(id),         -- set for match messages
  match_request_id UUID REFERENCES match_requests(id), -- set for pre-accept messages
  sender_id UUID NOT NULL REFERENCES users(id),
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK ((match_id IS NOT NULL) <> (match_request_id IS NOT NULL))
)
```

When a connection request is accepted, pre-accept messages are re-associated from `match_request_id` to `match_id` (UPDATE, not copy). This means the full conversation history — including pre-accept messages — is available under the match.

### New Migration: 00008_messaging_indexes.sql

Performance indexes for the conversations hub and message queries:

```sql
-- +goose Up

-- Conversation list: find all active matches for a user, ordered by latest message
CREATE INDEX IF NOT EXISTS matches_user_a_idx ON matches(user_a) WHERE closed_at IS NULL;
CREATE INDEX IF NOT EXISTS matches_user_b_idx ON matches(user_b) WHERE closed_at IS NULL;

-- Latest message per match (for conversation list snippet)
-- The existing messages_match_idx covers (match_id, created_at) which is sufficient
-- for ordering messages within a match. No additional index needed.

-- +goose Down
DROP INDEX IF EXISTS matches_user_b_idx;
DROP INDEX IF EXISTS matches_user_a_idx;
```

### Why No New Tables

- The `messages` table is already designed for match messaging (the `match_id` FK).
- No read receipts, typing indicators, or message status tracking in V1 — those would need new columns/tables.
- The `matches` table already has `closed_at` for soft-close (unmatch).

## Endpoints

### GET /api/v1/matches

List the authenticated user's active conversations (accepted matches where `closed_at IS NULL`).

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | null | Opaque pagination cursor |
| `limit` | int | 20 | Page size (max 50) |

**Response (200):**

```json
{
  "items": [
    {
      "match_id": "uuid",
      "partner": {
        "user_id": "uuid",
        "handle": "maria",
        "country_code": "MX",
        "age": 24
      },
      "last_message": {
        "body": "See you tomorrow at 3!",
        "sender_id": "uuid",
        "created_at": "2026-05-18T14:30:00Z"
      },
      "created_at": "2026-04-06T10:00:00Z"
    }
  ],
  "next_cursor": "..."
}
```

**Behavior:**

- Returns matches where the authenticated user is `user_a` or `user_b`.
- Ordered by `last_message.created_at DESC` (most recent conversation first). Matches with no messages yet sort by `matches.created_at DESC`.
- Each item includes the partner's profile summary and the last message (if any).
- Only active matches (`closed_at IS NULL`).

**Implementation Notes:**

- Query: join `matches` with a lateral subquery for the latest message, join `profiles` for the partner.
- Cursor is based on `(last_message_at, match_id)` for stable pagination.

### GET /api/v1/matches/{id}/messages

Paginated message history for a specific match.

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | null | Opaque cursor (for older messages) |
| `limit` | int | 50 | Page size (max 100) |

**Response (200):**

```json
{
  "items": [
    {
      "id": "uuid",
      "sender_id": "uuid",
      "body": "Hi! Great to connect!",
      "created_at": "2026-04-06T10:05:00Z"
    }
  ],
  "next_cursor": "..."
}
```

**Behavior:**

- Messages ordered by `created_at DESC` (newest first). The UI reverses for display.
- Includes re-associated pre-accept messages (full conversation history since first contact).
- Cursor-based pagination for loading older messages.
- Match must be active (`closed_at IS NULL`) and the authenticated user must be a participant.

### POST /api/v1/matches/{id}/messages

Send a message in an active match conversation.

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `body` | string | Yes | Message text (max 2000 chars) |

**Response (201):**

```json
{
  "id": "uuid",
  "sender_id": "uuid",
  "body": "Want to practice Thursday evening?",
  "created_at": "2026-05-18T14:31:00Z"
}
```

**Behavior:**

- Validates: match exists, is active, user is a participant.
- Body max length: 2000 characters (configurable via `MATCH_MESSAGE_MAX_LENGTH`).
- Daily message limit per user per match: configurable via `MATCH_DAILY_MESSAGE_LIMIT` (default 200). Prevents spam while allowing natural conversation.

### POST /api/v1/matches/{id}/close

Close (unmatch) a match. Either participant can close.

**Response (200):**

```json
{ "ok": true }
```

**Behavior:**

- Sets `closed_at = now()` on the match row.
- After closing, no new messages can be sent. Existing messages remain readable (for future reference, not V1).
- V1: closed matches are simply hidden from the conversations list.

## Error Codes

| Code | Condition |
|------|-----------|
| `ERR_MATCH_NOT_FOUND` | Match does not exist or user is not a participant |
| `ERR_MATCH_CLOSED` | Match has been closed (unmatch) |
| `ERR_MESSAGE_TOO_LONG` | Body exceeds max length |
| `ERR_DAILY_MESSAGE_LIMIT` | Daily message limit reached for this match |

All error messages localized via `Accept-Language` across all 11 supported locales.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MATCH_MESSAGE_MAX_LENGTH` | 2000 | Max message body length (chars) |
| `MATCH_DAILY_MESSAGE_LIMIT` | 200 | Max messages per user per match per day |

## Real-Time Update Strategy

### V1: Polling

For V1, the UI uses short polling to fetch new messages:

- **Chat view:** Poll `GET /matches/{id}/messages` every 3–5 seconds while the chat is open. Use `since` parameter (timestamp of latest known message) to fetch only new messages.
- **Conversations hub:** Poll `GET /matches` every 15–30 seconds while the list is visible.
- **No polling when tab is hidden:** Use `document.visibilitychange` to pause polling.

**Why polling for V1:**

- Simplest to implement and debug.
- No new infrastructure (no WebSocket server, no Redis pub/sub).
- Adequate for the expected V1 user volume.
- Pre-accept messaging already uses this pattern (fetch on mount + manual refresh).

### New Query Parameter for Polling Efficiency

Add a `since` query parameter to `GET /matches/{id}/messages`:

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `since` | string (ISO-8601) | null | Only return messages created after this timestamp |

When `since` is provided:
- Return messages with `created_at > since`, ordered `ASC` (oldest first for appending).
- No cursor pagination (the result set should be small for frequent polls).
- If no new messages, return `{"items": []}`.

When `since` is omitted, the endpoint behaves as the standard paginated history (DESC order, cursor-based).

### Future: Server-Sent Events (V2)

When polling becomes insufficient, upgrade to SSE:

- `GET /api/v1/matches/{id}/events` — SSE stream for a specific match.
- Events: `message.new`, `message.deleted`, `typing.start`, `typing.stop`.
- Server-side: per-match channels via in-process pub/sub (single-instance) or Redis pub/sub (multi-instance).
- The polling endpoints remain as fallback and for initial data load.

This is deferred to V2 to avoid premature infrastructure complexity.

## Architecture

Following the established layer separation:

| Layer | File | Responsibility |
|-------|------|---------------|
| Handler | `internal/http/messaging.go` | Route registration, input parsing, response formatting |
| Service | `internal/service/messaging.go` | Authorization (match membership), daily limits, validation |
| Repository | `internal/repository/messaging.go` | SQL queries: list matches with latest message, CRUD messages, daily count |

### Route Registration

```go
func registerMessagingRoutes(api huma.API, cfg config.Config, pool *pgxpool.Pool) {
    h := &messagingHandler{svc: service.NewMessagingService(cfg, repository.NewMessagingRepository(pool))}
    huma.Get(api, "/matches", h.listMatches)
    huma.Get(api, "/matches/{id}/messages", h.listMessages)
    huma.Post(api, "/matches/{id}/messages", h.sendMessage)
    huma.Post(api, "/matches/{id}/close", h.closeMatch)
}
```

### Key Repository Queries

**List matches with last message (conversations hub):**

```sql
SELECT m.id, m.user_a, m.user_b, m.created_at,
       msg.body AS last_message_body,
       msg.sender_id AS last_message_sender_id,
       msg.created_at AS last_message_at
FROM matches m
LEFT JOIN LATERAL (
    SELECT body, sender_id, created_at
    FROM messages
    WHERE match_id = m.id
    ORDER BY created_at DESC
    LIMIT 1
) msg ON true
WHERE (m.user_a = $1 OR m.user_b = $1)
  AND m.closed_at IS NULL
ORDER BY COALESCE(msg.created_at, m.created_at) DESC
LIMIT $2;
```

**Fetch new messages (polling with `since`):**

```sql
SELECT id, sender_id, body, created_at
FROM messages
WHERE match_id = $1
  AND created_at > $2
ORDER BY created_at ASC;
```

**Daily message count (rate limiting):**

```sql
SELECT COUNT(*)
FROM messages
WHERE match_id = $1
  AND sender_id = $2
  AND created_at >= (now() AT TIME ZONE 'UTC')::date;
```

## i18n

New error codes to add across all 11 locale files:

| Key | English (en) |
|-----|-------------|
| `ERR_MATCH_NOT_FOUND` | "Conversation not found." |
| `ERR_MATCH_CLOSED` | "This conversation has been closed." |
| `ERR_MESSAGE_TOO_LONG` | "Message is too long (max {max} characters)." |
| `ERR_DAILY_MESSAGE_LIMIT` | "You've reached your daily message limit for this conversation." |

## Migration Summary

- **00008_messaging_indexes.sql**: Add indexes on `matches(user_a)` and `matches(user_b)` filtered by `closed_at IS NULL` for efficient conversation listing. No new tables or columns.

## Testing Plan

- **Unit tests (service):** Authorization checks (participant vs non-participant), daily limit enforcement, closed match rejection, message length validation.
- **Integration tests (repository):** Conversation listing with/without messages, pagination, `since` parameter filtering, daily count queries.
- **HTTP tests (handler):** Full request/response cycle for all 4 endpoints, error codes, i18n.
- **E2E:** Extend `scripts/e2e-test.py` with messaging scenarios: send messages in accepted match, verify conversation list, test daily limits, close match, verify closed match blocks new messages.
