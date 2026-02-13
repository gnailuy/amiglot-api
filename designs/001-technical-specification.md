# Amiglot API — Technical Specification (Backend)

## 1. Technical Constraints
- Go 1.24
- Huma (HTTP framework)
- PostgreSQL with pgx + sqlc, migrations via goose
- API port: 6174

> Shared UI ↔ API contract lives in `amiglot-ui/designs/003-technical-specification.md`.

## 2. Database Schema (V1)
This section defines the **database schema** for V1.

### 2.1 Conventions
- **Primary keys:** UUID (`gen_random_uuid()`)
- **Timestamps:** `timestamptz` in UTC
- **Handles:** stored **without** `@`, UI displays with `@`
- **Timezone:** IANA name (e.g., `America/Vancouver`)
- **Languages:** BCP-47 language code (e.g., `en`, `es-ES`)

### 2.2 Core Tables

**users**
Auth + identity (email only in V1).

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_login_at TIMESTAMPTZ
);
```

**profiles**
One row per user.

```sql
CREATE TABLE profiles (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  handle TEXT NOT NULL UNIQUE,
  handle_norm TEXT NOT NULL UNIQUE,
  birth_year INT,
  birth_month SMALLINT CHECK (birth_month BETWEEN 1 AND 12),
  country_code CHAR(2),
  timezone TEXT NOT NULL,
  discoverable BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (handle ~ '^[a-zA-Z0-9_]+$')
);

-- keep handle_norm in lowercase (app-side or trigger)
```

> Notes
> - `handle_norm` is the lowercase version of `handle` for case-insensitive uniqueness.
> - `discoverable` is set by the app when minimum profile + language rules are satisfied.

**user_languages**
All languages a user knows and/or wants to learn.

```sql
CREATE TABLE user_languages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  language_code TEXT NOT NULL,
  level SMALLINT NOT NULL CHECK (level BETWEEN 0 AND 5),
  is_native BOOLEAN NOT NULL DEFAULT false,
  is_target BOOLEAN NOT NULL DEFAULT false,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, language_code)
);

CREATE INDEX user_languages_user_id_idx ON user_languages(user_id);
CREATE INDEX user_languages_language_idx ON user_languages(language_code, level);
```

> Rules enforced by the app:
> - At least one `is_native = true` per user
> - Target languages can overlap with native/teachable languages but do not have to

**availability_slots**
Weekly availability stored in **local time + timezone** (no static UTC columns). Matching converts to UTC for specific dates at query time to handle DST shifts correctly.

```sql
CREATE TABLE availability_slots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
  start_local_time TIME NOT NULL,
  end_local_time TIME NOT NULL,
  timezone TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX availability_user_idx ON availability_slots(user_id);
CREATE INDEX availability_local_idx ON availability_slots(weekday, start_local_time, end_local_time);
```

> Notes
> - App ensures `start < end` and handles wrap-around by splitting into two rows.
> - `timezone` defaults to the profile timezone but can be overridden per slot.
> - Matching converts `(date + weekday + start/end_local_time AT TIME ZONE timezone)` into UTC during search.

### 2.3 Matching & Messaging

**match_requests**

```sql
CREATE TABLE match_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('pending','accepted','declined','canceled')),
  intro_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  responded_at TIMESTAMPTZ
);

-- Only one pending request between two users
CREATE UNIQUE INDEX match_requests_unique_pending
  ON match_requests (requester_id, recipient_id)
  WHERE status = 'pending';
```

**matches**

```sql
CREATE TABLE matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_a UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_b UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at TIMESTAMPTZ,
  CHECK (user_a <> user_b)
);

-- prevent duplicate pairs (order-independent)
CREATE UNIQUE INDEX matches_unique_pair
  ON matches (LEAST(user_a, user_b), GREATEST(user_a, user_b));
```

**messages**

```sql
CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX messages_match_idx ON messages(match_id, created_at);
```

> Application ensures `sender_id` belongs to the match.

### 2.4 Safety & Admin (Minimal V1)

**user_blocks**

```sql
CREATE TABLE user_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (blocker_id, blocked_id)
);
```

**user_reports**

```sql
CREATE TABLE user_reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 2.5 Query Examples (Use Cases)
Pseudo-SQL showing the logic; actual implementation can be optimized with CTEs and indexes.

**Search candidates (filters + mutual match rules)**
```sql
-- Inputs: :user_id, :target_language, :min_level, :age_range, :country_code
WITH me AS (
  SELECT id FROM users WHERE id = :user_id
),
my_teach AS (
  SELECT language_code
  FROM user_languages
  WHERE user_id = :user_id AND level >= 4
),
my_targets AS (
  SELECT language_code
  FROM user_languages
  WHERE user_id = :user_id AND is_target = true
),
my_bridge AS (
  SELECT language_code
  FROM user_languages
  WHERE user_id = :user_id AND level >= 3
)
SELECT u.id
FROM users u
JOIN profiles p ON p.user_id = u.id
WHERE p.discoverable = true
  AND (:country_code IS NULL OR p.country_code = :country_code)
  AND u.id <> :user_id
  AND EXISTS (
    SELECT 1 FROM user_languages ul
    WHERE ul.user_id = u.id
      AND ul.language_code IN (SELECT language_code FROM my_targets)
      AND ul.level >= 4
  )
  AND EXISTS (
    SELECT 1 FROM user_languages ul
    WHERE ul.user_id = u.id
      AND ul.language_code IN (SELECT language_code FROM my_teach)
      AND ul.level >= 4
  )
  AND EXISTS (
    SELECT 1 FROM user_languages ul
    WHERE ul.user_id = u.id
      AND ul.language_code IN (SELECT language_code FROM my_bridge)
      AND ul.level >= 3
  );
```

**Create match request**
```sql
INSERT INTO match_requests (requester_id, recipient_id, status, intro_message)
VALUES (:requester_id, :recipient_id, 'pending', :intro_message);
```

**Accept match request**
```sql
UPDATE match_requests
SET status = 'accepted', responded_at = now()
WHERE id = :request_id AND recipient_id = :user_id;

INSERT INTO matches (user_a, user_b)
VALUES (:requester_id, :recipient_id)
ON CONFLICT DO NOTHING;
```

**Send message**
```sql
INSERT INTO messages (match_id, sender_id, body)
VALUES (:match_id, :sender_id, :body);
```

### 2.6 Migration Notes
- Existing `users` table already present in `amiglot-api` migrations; add new tables via sequential migrations.
- When user changes handle, update `profiles.handle` and `profiles.handle_norm`.
- Availability slots are stored in local time + timezone; matching converts to UTC at query time, so DST shifts are handled without rewriting rows.
