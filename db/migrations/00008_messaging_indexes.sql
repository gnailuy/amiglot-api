-- +goose Up

-- Partial indexes on matches for efficient conversation listing (active matches only)
CREATE INDEX IF NOT EXISTS matches_user_a_active_idx
    ON matches(user_a)
    WHERE closed_at IS NULL;

CREATE INDEX IF NOT EXISTS matches_user_b_active_idx
    ON matches(user_b)
    WHERE closed_at IS NULL;

-- Index for message polling with since parameter (ASC order)
CREATE INDEX IF NOT EXISTS messages_match_created_asc_idx
    ON messages(match_id, created_at ASC)
    WHERE match_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS messages_match_created_asc_idx;
DROP INDEX IF EXISTS matches_user_b_active_idx;
DROP INDEX IF EXISTS matches_user_a_active_idx;
