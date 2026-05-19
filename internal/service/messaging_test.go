package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/repository"
)

func createMessagingTestMatch(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userA, userB string) string {
	t.Helper()
	var matchID string
	err := pool.QueryRow(ctx,
		`INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid)) RETURNING id`,
		userA, userB,
	).Scan(&matchID)
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	return matchID
}

func TestMessagingService_ListConversations(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewMessagingRepository(pool)
	svc := NewMessagingService(repo, 2000, 200)

	userA := createConnTestUser(t, ctx, pool, "msg-list-a@test.com", "msglista")
	userB := createConnTestUser(t, ctx, pool, "msg-list-b@test.com", "msglistb")

	// No conversations
	result, err := svc.ListConversations(ctx, userA, 20, 0)
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}

	// Create a match and send a message
	matchID := createMessagingTestMatch(t, ctx, pool, userA, userB)
	_, _, err = repository.NewMessagingRepository(pool).CreateMatchMessage(ctx, matchID, userA, "Hello!")
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	result, err = svc.ListConversations(ctx, userA, 20, 0)
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].PartnerHandle != "msglistb" {
		t.Errorf("expected partner handle msglistb, got %s", result.Items[0].PartnerHandle)
	}
	if result.Items[0].LastMessageBody == nil || *result.Items[0].LastMessageBody != "Hello!" {
		t.Errorf("expected last message 'Hello!', got %v", result.Items[0].LastMessageBody)
	}

	// Unauthorized
	_, err = svc.ListConversations(ctx, "", 20, 0)
	assertServiceError(t, err, 401, "errors.missing_user_id")
}

func TestMessagingService_SendMessage(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewMessagingRepository(pool)
	svc := NewMessagingService(repo, 2000, 200)

	userA := createConnTestUser(t, ctx, pool, "msg-send-a@test.com", "msgsenda")
	userB := createConnTestUser(t, ctx, pool, "msg-send-b@test.com", "msgsendb")
	matchID := createMessagingTestMatch(t, ctx, pool, userA, userB)

	// Send message
	result, err := svc.SendMessage(ctx, matchID, userA, "Hi there!")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if result.Body != "Hi there!" {
		t.Errorf("expected 'Hi there!', got %s", result.Body)
	}
	if result.SenderID != userA {
		t.Errorf("expected sender %s, got %s", userA, result.SenderID)
	}

	// Empty body
	_, err = svc.SendMessage(ctx, matchID, userA, "")
	assertServiceError(t, err, 400, "errors.message_required")

	// Too long
	longMsg := make([]byte, 2001)
	for i := range longMsg {
		longMsg[i] = 'x'
	}
	_, err = svc.SendMessage(ctx, matchID, userA, string(longMsg))
	assertServiceError(t, err, 400, "errors.message_too_long")

	// Non-participant
	userC := createConnTestUser(t, ctx, pool, "msg-send-c@test.com", "msgsendc")
	_, err = svc.SendMessage(ctx, matchID, userC, "sneaky")
	assertServiceError(t, err, 404, "errors.match_not_found")

	// Unauthorized
	_, err = svc.SendMessage(ctx, matchID, "", "hi")
	assertServiceError(t, err, 401, "errors.missing_user_id")
}

func TestMessagingService_ListMessages(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewMessagingRepository(pool)
	svc := NewMessagingService(repo, 2000, 200)

	userA := createConnTestUser(t, ctx, pool, "msg-lm-a@test.com", "msglma")
	userB := createConnTestUser(t, ctx, pool, "msg-lm-b@test.com", "msglmb")
	matchID := createMessagingTestMatch(t, ctx, pool, userA, userB)

	// Send some messages
	for i := 0; i < 3; i++ {
		_, err := svc.SendMessage(ctx, matchID, userA, "msg")
		if err != nil {
			t.Fatalf("send: %v", err)
		}
	}

	// List messages (initial load, DESC)
	result, err := svc.ListMessages(ctx, matchID, userA, nil, nil, 50)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}

	// Non-participant
	userC := createConnTestUser(t, ctx, pool, "msg-lm-c@test.com", "msglmc")
	_, err = svc.ListMessages(ctx, matchID, userC, nil, nil, 50)
	assertServiceError(t, err, 404, "errors.match_not_found")
}

func TestMessagingService_CloseMatch(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewMessagingRepository(pool)
	svc := NewMessagingService(repo, 2000, 200)

	userA := createConnTestUser(t, ctx, pool, "msg-close-a@test.com", "msgclosea")
	userB := createConnTestUser(t, ctx, pool, "msg-close-b@test.com", "msgcloseb")
	matchID := createMessagingTestMatch(t, ctx, pool, userA, userB)

	// Close match
	err := svc.CloseMatch(ctx, matchID, userA)
	if err != nil {
		t.Fatalf("close match: %v", err)
	}

	// Double close
	err = svc.CloseMatch(ctx, matchID, userA)
	assertServiceError(t, err, 409, "errors.match_closed")

	// Can't send after close
	_, err = svc.SendMessage(ctx, matchID, userA, "hello")
	assertServiceError(t, err, 403, "errors.match_closed")

	// Non-participant
	userC := createConnTestUser(t, ctx, pool, "msg-close-c@test.com", "msgclosec")
	matchID2 := createMessagingTestMatch(t, ctx, pool, userA, userC)
	err = svc.CloseMatch(ctx, matchID2, userB)
	assertServiceError(t, err, 404, "errors.match_not_found")
}

func TestMessagingService_DailyLimit(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := repository.NewMessagingRepository(pool)
	// Low limit for testing
	svc := NewMessagingService(repo, 2000, 3)

	userA := createConnTestUser(t, ctx, pool, "msg-limit-a@test.com", "msglimita")
	userB := createConnTestUser(t, ctx, pool, "msg-limit-b@test.com", "msglimitb")
	matchID := createMessagingTestMatch(t, ctx, pool, userA, userB)

	// Send up to limit
	for i := 0; i < 3; i++ {
		_, err := svc.SendMessage(ctx, matchID, userA, "hi")
		if err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	// Next should fail
	_, err := svc.SendMessage(ctx, matchID, userA, "over limit")
	assertServiceError(t, err, 429, "errors.daily_message_limit")

	// Other user can still send
	_, err = svc.SendMessage(ctx, matchID, userB, "still ok")
	if err != nil {
		t.Fatalf("user B send: %v", err)
	}
}
