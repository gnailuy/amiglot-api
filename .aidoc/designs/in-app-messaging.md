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

Text messaging between connected (matched) partners. Adds match-scoped messaging endpoints on top of the existing `messages` table — no new tables required.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Database Schema](database-schema.md) | `messages`, `matches` tables |
| [Connection Handshake](connection-handshake.md) | Accept flow creates the match and re-associates pre-accept messages |
| [In-App Messaging (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/in-app-messaging.md) | Frontend conversations hub and chat interface |
| [Architecture Guidelines](../architecture/guidelines.md) | Layer separation, i18n, error handling |
| [API Contract](api-contract.md) | Shared endpoint conventions |

## Why This Design Exists

With connections in place, users need a way to communicate. Pre-accept messaging already proves the `messages` table works for match-scoped conversations. This design extends that pattern to post-accept messaging without schema sprawl.

## What This Design Covers

### Data Model Decision

The existing `messages` table already has a `match_id` FK. When a connection is accepted, pre-accept messages are re-associated (UPDATE, not copy) from `match_request_id` to `match_id`, giving full conversation continuity. No new tables or columns are needed in V1 — no read receipts, typing indicators, or message status tracking.

One migration (`00008_messaging_indexes.sql`) adds partial indexes on `matches(user_a)` and `matches(user_b)` filtered by `closed_at IS NULL` for efficient conversation listing.

### Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/matches` | Conversations list with partner info and last message snippet |
| `GET` | `/matches/{id}/messages` | Paginated history; supports `since` param for polling |
| `POST` | `/matches/{id}/messages` | Send message (max 2000 chars, 200/day per match) |
| `POST` | `/matches/{id}/close` | Soft close (unmatch); sets `closed_at`, blocks new messages |

The conversations list uses a lateral subquery for the latest message per match. See `repository.MessagingRepository.ListMatches` for the query.

The `since` parameter changes behavior: when present, returns messages created after that timestamp in ASC order (for appending during polling); when absent, returns standard DESC-ordered paginated history.

### Real-Time Strategy

**V1: Short polling.** The UI polls the messages endpoint every 3–5 seconds (chat) and the matches endpoint every 15–30 seconds (hub). Polling pauses when the browser tab is hidden. This avoids new infrastructure (no WebSocket server, no Redis pub/sub) and is adequate for V1 user volume.

**V2 (deferred):** SSE via `GET /matches/{id}/events` with per-match channels. Documented in code comments but not implemented — avoids premature infra complexity.

### Error Codes and Configuration

| Error Code | Condition |
|------------|-----------|
| `ERR_MATCH_NOT_FOUND` | Match does not exist or user is not a participant |
| `ERR_MATCH_CLOSED` | Match has been closed |
| `ERR_MESSAGE_TOO_LONG` | Body exceeds `MATCH_MESSAGE_MAX_LENGTH` (default 2000) |
| `ERR_DAILY_MESSAGE_LIMIT` | Exceeds `MATCH_DAILY_MESSAGE_LIMIT` (default 200) per user per match per day |

All error messages localized via `Accept-Language` across all 11 supported locales.

### Key Decisions

- **2000 char limit** (up from 500 for pre-accept) — connected users have higher trust.
- **200 messages/day** per match — high enough for natural conversation, low enough to prevent abuse.
- **Polling over WebSockets** — simplest V1 approach; SSE upgrade path documented for V2.
- **Closed matches hide from list** — messages remain readable in V1 but UI doesn't expose this yet.

## How It Works

Follows the standard handler → service → repository layering. See `internal/handler/messaging.go` for route registration, `internal/service/messaging.go` for authorization and rate-limit checks, and `internal/repository/messaging.go` for SQL queries.

<!-- TODO: (arturo) update entry_points after implementation to reflect actual file paths if they differ -->
