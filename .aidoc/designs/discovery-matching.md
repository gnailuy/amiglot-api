---
domain: Designs
status: Active
entry_points:
  - internal/http/discovery.go
  - internal/service/discovery.go
  - internal/repository/discovery.go
dependencies:
  - .aidoc/designs/database-schema.md
  - .aidoc/architecture/guidelines.md
---

# Discovery & Matching — Design

Design for `GET /api/v1/matches/discover` — the primary discovery endpoint returning paginated potential language exchange partners.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Database Schema](database-schema.md) | Tables queried by the matching CTE |
| [Architecture Guidelines](../architecture/guidelines.md) | Handler → Service → Repository layers |
| [Discovery Dashboard (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/discovery-dashboard.md) | UI dashboard that consumes these matching results |
| [API Contract](api-contract.md) | Shared endpoint conventions |
| [Matching Query](discovery-matching-query.md) | Full SQL CTE and index strategy |
| [Connection Handshake](connection-handshake.md) | Next step after discovery |

## Why This Endpoint Exists

Discovery is the primary user-facing surface for finding language exchange partners. The matching algorithm enforces mutual exchange (supply + demand + bridge checks) plus availability overlap, ensuring meaningful matches.

## Endpoint Contract

```
GET /api/v1/matches/discover
Authorization: Bearer <token>
Accept-Language: <locale>
```

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
      "user_id": "uuid",
      "handle": "maria",
      "country_code": "MX",
      "age": 24,
      "mutual_teach": [
        { "language_code": "es", "level": 5, "is_native": true }
      ],
      "mutual_learn": [
        { "language_code": "en", "level": 4, "is_native": false }
      ],
      "bridge_languages": [
        { "language_code": "en", "level": 4 }
      ],
      "availability_overlap": [
        {
          "weekday": 1,
          "start_utc": "01:00",
          "end_utc": "03:00",
          "overlap_minutes": 120
        }
      ],
      "total_overlap_minutes": 120
    }
  ],
  "next_cursor": "..."
}
```

## Matching Rules (All Must Pass)

1. **Supply check** — Candidate teaches what user wants at level ≥ 4
2. **Demand check** — User teaches what candidate wants at level ≥ 4
3. **Bridge check** — Shared language where both are level ≥ 3
4. **Availability overlap** — At least `MATCH_MIN_OVERLAP_MINUTES` (default 60) shared weekly time
5. **Discoverable** — Candidate has `profiles.discoverable = true`
6. **Not blocked** — Neither user has blocked the other
7. **Not self** — Exclude requesting user

## Architecture

- **Handler** (`internal/handler/discovery.go`): parse query params, extract user from auth, call service, serialize
- **Service** (`internal/service/discovery.go`): orchestrate matching — call repository, compute overlaps, apply business rules, paginate
- **Repository** (`internal/repository/discovery.go`): execute the SQL matching CTE

## Response Assembly (Service Layer)

After SQL returns candidate IDs + overlap totals:
1. Batch-fetch candidate languages
2. Compute `mutual_teach`, `mutual_learn`, `bridge_languages` by intersecting with requesting user's data
3. Fetch per-slot overlap details
4. Compute age from `birth_year`/`birth_month`
5. Assemble response objects

## Error Handling

| Status | Code | Condition |
|--------|------|-----------|
| 401 | `ERR_AUTH_REQUIRED` | Missing/invalid token |
| 403 | `ERR_PROFILE_INCOMPLETE` | User not discoverable |
| 422 | `ERR_NO_TARGET_LANGUAGES` | No target languages set |

## Configuration

- `MATCH_MIN_OVERLAP_MINUTES` (default 60) — configurable via env var
