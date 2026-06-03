---
domain: Designs
status: Active
entry_points:
  - internal/http/connection.go
  - internal/service/connection.go
  - internal/repository/connection.go
dependencies:
  - .aidoc/designs/database-schema.md
  - .aidoc/designs/discovery-matching.md
  - .aidoc/architecture/guidelines.md
---

# Connection (Handshake) — Design

State machine for connection requests: `None → Pending → Accepted/Declined/Canceled`. Covers sending requests, inbox listing, accept/decline/cancel actions, and pre-accept messaging.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Database Schema](database-schema.md) | match_requests, matches, messages tables |
| [Discovery Matching](discovery-matching.md) | Discovery feeds into connection requests |
| [Connection Handshake (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/connection-handshake.md) | UI flows and components for this state machine |
| [Architecture Guidelines](../architecture/guidelines.md) | Layer separation pattern |
| [API Contract](api-contract.md) | Shared endpoint conventions |

## Why This Design Exists

The handshake ensures mutual consent before connecting users. Pre-accept messaging lets users evaluate compatibility before committing, while message limits prevent spam.

## Endpoints

### POST /api/v1/match-requests

Send a connection request. Validates: recipient exists and is discoverable, not self, no pending request, no existing match, not blocked.

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `recipient_id` | uuid | Yes | Target user ID |
| `initial_message` | string | No | Optional message (max 500 chars) |

**Response (201):**

```json
{
  "id": "uuid",
  "requester_id": "uuid",
  "recipient_id": "uuid",
  "status": "pending",
  "initial_message": "Hi! I'd love to practice Spanish with you.",
  "created_at": "2026-03-28T12:00:00Z"
}
```

### GET /api/v1/match-requests

List connection requests (incoming and outgoing).

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `direction` | string | `incoming` | `incoming` or `outgoing` |
| `status` | string | `pending` | `pending`, `accepted`, `declined`, `canceled`, or `all` |
| `cursor` | string | null | Opaque pagination cursor |
| `limit` | int | 20 | Page size (max 50) |

**Response (200):**

```json
{
  "items": [
    {
      "id": "uuid",
      "requester": {
        "user_id": "uuid",
        "handle": "maria",
        "country_code": "MX",
        "age": 24
      },
      "recipient": {
        "user_id": "uuid",
        "handle": "arturo",
        "country_code": "CA",
        "age": 32
      },
      "status": "pending",
      "message_count": 1,
      "last_message_at": "2026-03-28T12:00:00Z",
      "created_at": "2026-03-28T12:00:00Z"
    }
  ],
  "next_cursor": "..."
}
```

### GET /api/v1/match-requests/{id}

Single request detail. Accessible by requester or recipient only.

**Response (200):** Same shape as a list item, plus:

```json
{
  "mutual_teach": [...],
  "mutual_learn": [...],
  "bridge_languages": [...]
}
```

### POST /api/v1/match-requests/{id}/accept

Recipient-only. Transactional: update status → create match (LEAST/GREATEST ordering) → re-associate messages from `match_request_id` to `match_id`.

**Response (200):**

```json
{ "ok": true, "match_id": "uuid" }
```

### POST /api/v1/match-requests/{id}/decline

Recipient-only. Sets status to `declined`.

**Response (200):** `{ "ok": true }`

### POST /api/v1/match-requests/{id}/cancel

Requester-only. Sets status to `canceled`.

**Response (200):** `{ "ok": true }`

### GET /api/v1/match-requests/{id}/messages

List pre-accept messages. Accessible by requester or recipient.

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
      "id": "uuid",
      "sender_id": "uuid",
      "body": "Hi! I'd love to practice Spanish with you.",
      "created_at": "2026-03-28T12:00:00Z"
    }
  ],
  "next_cursor": "..."
}
```

### POST /api/v1/match-requests/{id}/messages

Send a pre-accept message. Per-user message limit enforced (`PRE_MATCH_MESSAGE_LIMIT`, default 5).

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `body` | string | Yes | Message text (max 500 chars) |

**Response (201):**

```json
{
  "id": "uuid",
  "sender_id": "uuid",
  "body": "Hello! When are you usually free?",
  "created_at": "2026-03-28T12:01:00Z"
}
```

## Error Codes

| Code | Condition |
|------|-----------|
| `ERR_SELF_REQUEST` | Requester = recipient |
| `ERR_USER_NOT_FOUND` | Recipient missing or not discoverable |
| `ERR_REQUEST_EXISTS` | Pending request already exists |
| `ERR_ALREADY_MATCHED` | Active match exists |
| `ERR_USER_BLOCKED` | Either user blocked the other |
| `ERR_NOT_RECIPIENT` | Caller is not the recipient |
| `ERR_NOT_PENDING` | Request not in pending status |
| `ERR_NOT_PARTICIPANT` | Sender not part of this request |
| `ERR_MESSAGE_LIMIT` | Pre-accept message limit reached |

All error messages localized via `Accept-Language` across all 11 supported locales.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PRE_MATCH_MESSAGE_LIMIT` | 5 | Max messages per user per request |
| `MATCH_REQUEST_MESSAGE_MAX_LENGTH` | 500 | Max body length |

## Architecture

- **Handler** (`internal/handler/connection.go`): route registration, input parsing
- **Service** (`internal/service/connection.go`): precondition validation, accept transaction, message limits
- **Repository** (`internal/repository/connection.go`): sqlc queries for CRUD, transactional accept

## Migration

Migration `00007_connection_handshake.sql` adds performance indexes for inbox queries and message count subqueries. No new tables — `match_requests`, `matches`, `messages` already exist.
