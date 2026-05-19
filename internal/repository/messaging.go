package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MessagingRepository handles match-scoped messaging queries.
type MessagingRepository struct {
	pool *pgxpool.Pool
}

// NewMessagingRepository creates a new MessagingRepository.
func NewMessagingRepository(pool *pgxpool.Pool) *MessagingRepository {
	return &MessagingRepository{pool: pool}
}

// ConversationRow represents a match (conversation) with partner info and last message.
type ConversationRow struct {
	ID                string
	PartnerID         string
	PartnerHandle     string
	PartnerCountry    *string
	PartnerBirthYear  *int
	PartnerBirthMonth *int16
	CreatedAt         time.Time
	ClosedAt          *time.Time
	LastMessageBody   *string
	LastMessageAt     *time.Time
	LastSenderID      *string
}

// ConvMessageRow represents a message in a match conversation.
type ConvMessageRow struct {
	ID        string
	SenderID  string
	Body      string
	CreatedAt time.Time
}

const listMatchesSQL = `
SELECT
    m.id, m.created_at, m.closed_at,
    CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS partner_id,
    p.handle AS partner_handle,
    p.country_code AS partner_country,
    p.birth_year AS partner_birth_year,
    p.birth_month AS partner_birth_month,
    lm.body AS last_message_body,
    lm.created_at AS last_message_at,
    lm.sender_id AS last_sender_id
FROM matches m
JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
LEFT JOIN LATERAL (
    SELECT body, created_at, sender_id
    FROM messages
    WHERE match_id = m.id
    ORDER BY created_at DESC
    LIMIT 1
) lm ON true
WHERE (m.user_a = $1 OR m.user_b = $1)
  AND m.closed_at IS NULL
ORDER BY COALESCE(lm.created_at, m.created_at) DESC
LIMIT $2 OFFSET $3
`

// ListMatches returns active matches for a user with last message info.
func (r *MessagingRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]ConversationRow, error) {
	rows, err := r.pool.Query(ctx, listMatchesSQL, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ConversationRow
	for rows.Next() {
		var m ConversationRow
		if err := rows.Scan(
			&m.ID, &m.CreatedAt, &m.ClosedAt,
			&m.PartnerID, &m.PartnerHandle, &m.PartnerCountry,
			&m.PartnerBirthYear, &m.PartnerBirthMonth,
			&m.LastMessageBody, &m.LastMessageAt, &m.LastSenderID,
		); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetMatch returns a single match. Returns nil if not found.
func (r *MessagingRepository) GetMatch(ctx context.Context, matchID, userID string) (*ConversationRow, error) {
	var m ConversationRow
	err := r.pool.QueryRow(ctx, `
		SELECT
			m.id, m.created_at, m.closed_at,
			CASE WHEN m.user_a = $2 THEN m.user_b ELSE m.user_a END AS partner_id,
			p.handle AS partner_handle,
			p.country_code AS partner_country,
			p.birth_year AS partner_birth_year,
			p.birth_month AS partner_birth_month
		FROM matches m
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $2 THEN m.user_b ELSE m.user_a END
		WHERE m.id = $1
		  AND (m.user_a = $2 OR m.user_b = $2)
	`, matchID, userID).Scan(
		&m.ID, &m.CreatedAt, &m.ClosedAt,
		&m.PartnerID, &m.PartnerHandle, &m.PartnerCountry,
		&m.PartnerBirthYear, &m.PartnerBirthMonth,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// ListMatchMessages returns messages for a match with pagination.
// If since is provided, returns messages after that time in ASC order.
// Otherwise returns DESC-ordered messages with cursor pagination.
func (r *MessagingRepository) ListMatchMessages(ctx context.Context, matchID string, since *time.Time, cursor *string, limit int) ([]ConvMessageRow, error) {
	var rows pgx.Rows
	var err error

	if since != nil {
		// Polling mode: messages after `since` in ASC order
		rows, err = r.pool.Query(ctx, `
			SELECT id, sender_id, body, created_at
			FROM messages
			WHERE match_id = $1 AND created_at > $2
			ORDER BY created_at ASC
			LIMIT $3
		`, matchID, *since, limit)
	} else if cursor != nil {
		// Cursor pagination: older messages
		rows, err = r.pool.Query(ctx, `
			SELECT id, sender_id, body, created_at
			FROM messages
			WHERE match_id = $1
			  AND created_at < (SELECT created_at FROM messages WHERE id = $2)
			ORDER BY created_at DESC
			LIMIT $3
		`, matchID, *cursor, limit)
	} else {
		// Initial load: newest messages
		rows, err = r.pool.Query(ctx, `
			SELECT id, sender_id, body, created_at
			FROM messages
			WHERE match_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`, matchID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ConvMessageRow
	for rows.Next() {
		var m ConvMessageRow
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// CreateMatchMessage inserts a message into a match conversation.
func (r *MessagingRepository) CreateMatchMessage(ctx context.Context, matchID, senderID, body string) (string, time.Time, error) {
	var id string
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`INSERT INTO messages (match_id, sender_id, body)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		matchID, senderID, body,
	).Scan(&id, &createdAt)
	return id, createdAt, err
}

// CountDailyMessages counts messages sent by a user in a match today (UTC).
func (r *MessagingRepository) CountDailyMessages(ctx context.Context, matchID, senderID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM messages
		 WHERE match_id = $1 AND sender_id = $2
		   AND created_at >= date_trunc('day', now())`,
		matchID, senderID,
	).Scan(&count)
	return count, err
}

// CloseMatch sets closed_at on a match. Returns false if already closed or not found.
func (r *MessagingRepository) CloseMatch(ctx context.Context, matchID, userID string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE matches SET closed_at = now()
		 WHERE id = $1
		   AND (user_a = $2 OR user_b = $2)
		   AND closed_at IS NULL`,
		matchID, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
