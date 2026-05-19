package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMessagingRepository_FullFlow(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	repo := NewMessagingRepository(pool)

	// Create two users with profiles
	var userA, userB, userC string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('msg-repo-a@test.com') RETURNING id`).Scan(&userA)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable, country_code, birth_year, birth_month)
		VALUES ($1, 'msgrepoa', 'msgrepoa', 'UTC', true, 'US', 2000, 6)`, userA)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('msg-repo-b@test.com') RETURNING id`).Scan(&userB)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable, country_code, birth_year, birth_month)
		VALUES ($1, 'msgrepob', 'msgrepob', 'UTC', true, 'CN', 1995, 3)`, userB)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('msg-repo-c@test.com') RETURNING id`).Scan(&userC)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'msgrepoc', 'msgrepoc', 'UTC', true)`, userC)
	require.NoError(t, err)

	// Create a match between A and B
	var matchID string
	err = pool.QueryRow(ctx,
		`INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid)) RETURNING id`,
		userA, userB).Scan(&matchID)
	require.NoError(t, err)

	// GetMatch — participant
	match, err := repo.GetMatch(ctx, matchID, userA)
	require.NoError(t, err)
	require.NotNil(t, match)
	require.Equal(t, matchID, match.ID)
	require.Equal(t, userB, match.PartnerID)
	require.Equal(t, "msgrepob", match.PartnerHandle)
	require.NotNil(t, match.PartnerCountry)
	require.Equal(t, "CN", *match.PartnerCountry)
	require.Nil(t, match.ClosedAt)

	// GetMatch — non-participant
	match, err = repo.GetMatch(ctx, matchID, userC)
	require.NoError(t, err)
	require.Nil(t, match)

	// ListMatches — empty (no messages yet, but match exists)
	matches, err := repo.ListMatches(ctx, userA, 10, 0)
	require.NoError(t, err)
	require.Len(t, matches, 1) // match is visible even without messages

	// CreateMatchMessage
	msgID, msgCreatedAt, err := repo.CreateMatchMessage(ctx, matchID, userA, "Hello!")
	require.NoError(t, err)
	require.NotEmpty(t, msgID)
	require.False(t, msgCreatedAt.IsZero())

	// ListMatches — now has last message
	matches, err = repo.ListMatches(ctx, userA, 10, 0)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.NotNil(t, matches[0].LastMessageBody)
	require.Equal(t, "Hello!", *matches[0].LastMessageBody)
	require.NotNil(t, matches[0].LastSenderID)
	require.Equal(t, userA, *matches[0].LastSenderID)

	// ListMatchMessages — initial load (DESC)
	msgs, err := repo.ListMatchMessages(ctx, matchID, nil, nil, 50)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "Hello!", msgs[0].Body)

	// Send another message
	msg2ID, _, err := repo.CreateMatchMessage(ctx, matchID, userB, "Hi back!")
	require.NoError(t, err)
	require.NotEmpty(t, msg2ID)

	// ListMatchMessages — since (polling)
	since := msgCreatedAt
	msgs, err = repo.ListMatchMessages(ctx, matchID, &since, nil, 50)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "Hi back!", msgs[0].Body)

	// ListMatchMessages — cursor pagination
	msgs, err = repo.ListMatchMessages(ctx, matchID, nil, &msg2ID, 50)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "Hello!", msgs[0].Body) // Older message

	// CountDailyMessages
	count, err := repo.CountDailyMessages(ctx, matchID, userA)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	count, err = repo.CountDailyMessages(ctx, matchID, userB)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// CloseMatch
	ok, err := repo.CloseMatch(ctx, matchID, userA)
	require.NoError(t, err)
	require.True(t, ok)

	// Double close
	ok, err = repo.CloseMatch(ctx, matchID, userA)
	require.NoError(t, err)
	require.False(t, ok)

	// GetMatch still works after close (shows closed_at)
	match, err = repo.GetMatch(ctx, matchID, userA)
	require.NoError(t, err)
	require.NotNil(t, match)
	require.NotNil(t, match.ClosedAt)

	// ListMatches — closed match hidden
	matches, err = repo.ListMatches(ctx, userA, 10, 0)
	require.NoError(t, err)
	require.Len(t, matches, 0)
}

func TestMessagingRepository_ListMatchesPagination(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()
	repo := NewMessagingRepository(pool)

	// Create user with multiple matches
	var mainUser string
	err := pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ('msg-page-main@test.com') RETURNING id`).Scan(&mainUser)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
		VALUES ($1, 'msgpagemain', 'msgpagemain', 'UTC', true)`, mainUser)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		var partner string
		email := fmt.Sprintf("msg-page-%d-%d@test.com", i, time.Now().UnixNano())
		err = pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ($1) RETURNING id`, email).Scan(&partner)
		require.NoError(t, err)
		handle := fmt.Sprintf("msgpage%d%d", i, time.Now().UnixNano())
		_, err = pool.Exec(ctx, `INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable)
			VALUES ($1, $2, $2, 'UTC', true)`, partner, handle)
		require.NoError(t, err)

		var matchID string
		err = pool.QueryRow(ctx,
			`INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid)) RETURNING id`,
			mainUser, partner).Scan(&matchID)
		require.NoError(t, err)

		// Add a message to each match to ensure ordering
		_, _, err = repo.CreateMatchMessage(ctx, matchID, mainUser, "msg"+time.Now().Format("150405.000"))
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Paginate
	page1, err := repo.ListMatches(ctx, mainUser, 2, 0)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	page2, err := repo.ListMatches(ctx, mainUser, 2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)
}
