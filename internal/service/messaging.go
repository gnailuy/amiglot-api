package service

import (
	"context"
	"time"

	"github.com/gnailuy/amiglot-api/internal/repository"
)

const (
	defaultMatchMessageMaxLen    = 2000
	defaultMatchDailyMsgLimit    = 200
	defaultMatchMessagesPageSize = 50
	maxMatchMessagesPageSize     = 100
)

// MessagingService handles post-accept messaging business logic.
type MessagingService struct {
	repo       *repository.MessagingRepository
	msgMaxLen  int
	dailyLimit int
}

// NewMessagingService creates a new MessagingService.
func NewMessagingService(repo *repository.MessagingRepository, msgMaxLen, dailyLimit int) *MessagingService {
	if msgMaxLen <= 0 {
		msgMaxLen = defaultMatchMessageMaxLen
	}
	if dailyLimit <= 0 {
		dailyLimit = defaultMatchDailyMsgLimit
	}
	return &MessagingService{repo: repo, msgMaxLen: msgMaxLen, dailyLimit: dailyLimit}
}

// ConversationItem represents a match in the conversations list.
type ConversationItem struct {
	ID              string
	PartnerID       string
	PartnerHandle   string
	PartnerCountry  *string
	PartnerAge      *int
	CreatedAt       time.Time
	LastMessageBody *string
	LastMessageAt   *time.Time
	LastSenderID    *string
}

// ConversationListResult is the result of listing conversations.
type ConversationListResult struct {
	Items   []ConversationItem
	HasMore bool
}

// ListConversations returns the user's active conversations.
func (s *MessagingService) ListConversations(ctx context.Context, userID string, limit, offset int) (*ConversationListResult, error) {
	if userID == "" {
		return nil, &Error{Status: 401, Key: "errors.missing_user_id"}
	}
	if limit <= 0 || limit > maxMatchMessagesPageSize {
		limit = defaultMatchMessagesPageSize
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.repo.ListMatches(ctx, userID, limit+1, offset)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	items := make([]ConversationItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, ConversationItem{
			ID:              r.ID,
			PartnerID:       r.PartnerID,
			PartnerHandle:   r.PartnerHandle,
			PartnerCountry:  r.PartnerCountry,
			PartnerAge:      ageFromBirthYear(r.PartnerBirthYear, r.PartnerBirthMonth),
			CreatedAt:       r.CreatedAt,
			LastMessageBody: r.LastMessageBody,
			LastMessageAt:   r.LastMessageAt,
			LastSenderID:    r.LastSenderID,
		})
	}

	return &ConversationListResult{Items: items, HasMore: hasMore}, nil
}

// MatchMessage represents a message in a conversation.
type MatchMessage struct {
	ID        string
	SenderID  string
	Body      string
	CreatedAt time.Time
}

// MatchMessageListResult is the result of listing match messages.
type MatchMessageListResult struct {
	Items      []MatchMessage
	NextCursor *string
}

// ListMessages returns messages for a match.
func (s *MessagingService) ListMessages(ctx context.Context, matchID, userID string, since *time.Time, cursor *string, limit int) (*MatchMessageListResult, error) {
	if userID == "" {
		return nil, &Error{Status: 401, Key: "errors.missing_user_id"}
	}
	if limit <= 0 || limit > maxMatchMessagesPageSize {
		limit = defaultMatchMessagesPageSize
	}

	// Verify the user is a participant
	match, err := s.repo.GetMatch(ctx, matchID, userID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if match == nil {
		return nil, &Error{Status: 404, Key: "errors.match_not_found"}
	}

	rows, err := s.repo.ListMatchMessages(ctx, matchID, since, cursor, limit+1)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	var nextCursor *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1].ID
		nextCursor = &last
	}

	items := make([]MatchMessage, 0, len(rows))
	for _, r := range rows {
		items = append(items, MatchMessage{
			ID:        r.ID,
			SenderID:  r.SenderID,
			Body:      r.Body,
			CreatedAt: r.CreatedAt,
		})
	}

	return &MatchMessageListResult{Items: items, NextCursor: nextCursor}, nil
}

// SendMessageResult is the result of sending a message.
type SendMessageResult struct {
	ID        string
	SenderID  string
	Body      string
	CreatedAt time.Time
}

// SendMessage sends a message in a match conversation.
func (s *MessagingService) SendMessage(ctx context.Context, matchID, senderID, body string) (*SendMessageResult, error) {
	if senderID == "" {
		return nil, &Error{Status: 401, Key: "errors.missing_user_id"}
	}
	if len(body) == 0 {
		return nil, &Error{Status: 400, Key: "errors.message_required"}
	}
	if len(body) > s.msgMaxLen {
		return nil, &Error{Status: 400, Key: "errors.message_too_long"}
	}

	// Verify participation and match status
	match, err := s.repo.GetMatch(ctx, matchID, senderID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if match == nil {
		return nil, &Error{Status: 404, Key: "errors.match_not_found"}
	}
	if match.ClosedAt != nil {
		return nil, &Error{Status: 403, Key: "errors.match_closed"}
	}

	// Check daily limit
	count, err := s.repo.CountDailyMessages(ctx, matchID, senderID)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if count >= s.dailyLimit {
		return nil, &Error{Status: 429, Key: "errors.daily_message_limit"}
	}

	id, createdAt, err := s.repo.CreateMatchMessage(ctx, matchID, senderID, body)
	if err != nil {
		return nil, &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}

	return &SendMessageResult{
		ID:        id,
		SenderID:  senderID,
		Body:      body,
		CreatedAt: createdAt,
	}, nil
}

// CloseMatch closes a match (unmatch).
func (s *MessagingService) CloseMatch(ctx context.Context, matchID, userID string) error {
	if userID == "" {
		return &Error{Status: 401, Key: "errors.missing_user_id"}
	}

	// Verify participation
	match, err := s.repo.GetMatch(ctx, matchID, userID)
	if err != nil {
		return &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if match == nil {
		return &Error{Status: 404, Key: "errors.match_not_found"}
	}
	if match.ClosedAt != nil {
		return &Error{Status: 409, Key: "errors.match_closed"}
	}

	ok, err := s.repo.CloseMatch(ctx, matchID, userID)
	if err != nil {
		return &Error{Status: 500, Key: "errors.internal_server_error", Err: err}
	}
	if !ok {
		return &Error{Status: 409, Key: "errors.match_closed"}
	}

	return nil
}
