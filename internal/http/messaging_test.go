package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createMsgUser(t *testing.T, pool *pgxpool.Pool, email, handle string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email) VALUES ($1) RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = pool.Exec(context.Background(),
		`INSERT INTO profiles (user_id, handle, handle_norm, timezone, discoverable, birth_year, birth_month, country_code)
		 VALUES ($1, $2, $2, 'UTC', true, 2000, 6, 'US')`, id, handle)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	return id
}

func createMsgMatch(t *testing.T, pool *pgxpool.Pool, userA, userB string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid)) RETURNING id`,
		userA, userB).Scan(&id)
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	return id
}

func TestMessagingEndpoint_ListConversations(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createMsgUser(t, pool, "msg-http-a@test.com", "msghttpa")
	userB := createMsgUser(t, pool, "msg-http-b@test.com", "msghttpb")
	matchID := createMsgMatch(t, pool, userA, userB)

	// Send a message to have something in the list
	body := `{"body":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/messages", matchID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("send: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// List conversations
	req = httptest.NewRequest(http.MethodGet, "/api/v1/matches", nil)
	req.Header.Set("X-User-Id", userA)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Items   []json.RawMessage `json:"items"`
		HasMore bool              `json:"has_more"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(listResp.Items))
	}

	// No auth
	req = httptest.NewRequest(http.MethodGet, "/api/v1/matches", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no auth: expected 401, got %d", rec.Code)
	}
}

func TestMessagingEndpoint_SendAndListMessages(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createMsgUser(t, pool, "msg-http-c@test.com", "msghttpc")
	userB := createMsgUser(t, pool, "msg-http-d@test.com", "msghttpd")
	matchID := createMsgMatch(t, pool, userA, userB)

	// Send message
	body := `{"body":"Test message"}`
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/messages", matchID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("send: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var sendResp struct {
		ID        string `json:"id"`
		SenderID  string `json:"sender_id"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&sendResp); err != nil {
		t.Fatalf("decode send: %v", err)
	}
	if sendResp.Body != "Test message" {
		t.Errorf("expected 'Test message', got %s", sendResp.Body)
	}

	// List messages
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/matches/%s/messages", matchID), nil)
	req.Header.Set("X-User-Id", userA)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Items      []json.RawMessage `json:"items"`
		NextCursor *string           `json:"next_cursor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Items) != 1 {
		t.Errorf("expected 1 message, got %d", len(listResp.Items))
	}

	// Non-participant
	userC := createMsgUser(t, pool, "msg-http-e@test.com", "msghttpe")
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/matches/%s/messages", matchID), nil)
	req.Header.Set("X-User-Id", userC)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("non-participant: expected 404, got %d", rec.Code)
	}
}

func TestMessagingEndpoint_CloseMatch(t *testing.T) {
	pool := openTestPool(t)
	cfg := config.Load()
	mux := Router(cfg, pool)

	userA := createMsgUser(t, pool, "msg-http-f@test.com", "msghttpf")
	userB := createMsgUser(t, pool, "msg-http-g@test.com", "msghttpg")
	matchID := createMsgMatch(t, pool, userA, userB)

	// Close
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/close", matchID), nil)
	req.Header.Set("X-User-Id", userA)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("close: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Send after close should fail
	body := `{"body":"after close"}`
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/matches/%s/messages", matchID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userA)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("send after close: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
