package http

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/repository"
	"github.com/gnailuy/amiglot-api/internal/service"
)

type messagingHandler struct {
	svc *service.MessagingService
}

// --- Request / Response types ---

type listConversationsInput struct {
	UserID string `header:"X-User-Id"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

type conversationPayload struct {
	ID              string  `json:"id"`
	PartnerID       string  `json:"partner_id"`
	PartnerHandle   string  `json:"partner_handle"`
	PartnerCountry  *string `json:"partner_country,omitempty"`
	PartnerAge      *int    `json:"partner_age,omitempty"`
	CreatedAt       string  `json:"created_at"`
	LastMessageBody *string `json:"last_message_body,omitempty"`
	LastMessageAt   *string `json:"last_message_at,omitempty"`
	LastSenderID    *string `json:"last_sender_id,omitempty"`
}

type listConversationsResponse struct {
	Body struct {
		Items   []conversationPayload `json:"items"`
		HasMore bool                  `json:"has_more"`
	}
}

type listMatchMessagesInput struct {
	UserID  string `header:"X-User-Id"`
	MatchID string `path:"id"`
	Since   string `query:"since"`
	Cursor  string `query:"cursor"`
	Limit   int    `query:"limit"`
}

type matchMessagePayload struct {
	ID        string `json:"id"`
	SenderID  string `json:"sender_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type listMatchMessagesResponse struct {
	Body struct {
		Items      []matchMessagePayload `json:"items"`
		NextCursor *string               `json:"next_cursor"`
	}
}

type sendMatchMessageInput struct {
	UserID  string `header:"X-User-Id"`
	MatchID string `path:"id"`
	Body    struct {
		Body string `json:"body" required:"true"`
	}
}

type sendMatchMessageResponse struct {
	Body matchMessagePayload
}

type closeMatchInput struct {
	UserID  string `header:"X-User-Id"`
	MatchID string `path:"id"`
}

type closeMatchResponse struct {
	Body struct {
		Ok bool `json:"ok"`
	}
}

// registerMessagingRoutes registers in-app messaging endpoints.
func registerMessagingRoutes(api huma.API, cfg config.Config, pool *pgxpool.Pool) {
	repo := repository.NewMessagingRepository(pool)
	svc := service.NewMessagingService(repo, cfg.MatchMessageMaxLen, cfg.MatchDailyMsgLimit)
	h := &messagingHandler{svc: svc}

	huma.Get(api, "/matches", h.listConversations)
	huma.Get(api, "/matches/{id}/messages", h.listMessages)
	huma.Post(api, "/matches/{id}/messages", h.sendMessage)
	huma.Post(api, "/matches/{id}/close", h.closeMatch)
}

func (h *messagingHandler) listConversations(ctx context.Context, input *listConversationsInput) (*listConversationsResponse, error) {
	result, err := h.svc.ListConversations(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	items := make([]conversationPayload, 0, len(result.Items))
	for _, item := range result.Items {
		cp := conversationPayload{
			ID:             item.ID,
			PartnerID:      item.PartnerID,
			PartnerHandle:  item.PartnerHandle,
			PartnerCountry: item.PartnerCountry,
			PartnerAge:     item.PartnerAge,
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		}
		if item.LastMessageBody != nil {
			cp.LastMessageBody = item.LastMessageBody
		}
		if item.LastMessageAt != nil {
			t := item.LastMessageAt.Format(time.RFC3339)
			cp.LastMessageAt = &t
		}
		if item.LastSenderID != nil {
			cp.LastSenderID = item.LastSenderID
		}
		items = append(items, cp)
	}

	resp := &listConversationsResponse{}
	resp.Body.Items = items
	resp.Body.HasMore = result.HasMore
	return resp, nil
}

func (h *messagingHandler) listMessages(ctx context.Context, input *listMatchMessagesInput) (*listMatchMessagesResponse, error) {
	var since *time.Time
	if input.Since != "" {
		t, err := time.Parse(time.RFC3339Nano, input.Since)
		if err != nil {
			t2, err2 := time.Parse(time.RFC3339, input.Since)
			if err2 != nil {
				return nil, huma.Error400BadRequest("invalid since format, expected RFC3339")
			}
			t = t2
		}
		since = &t
	}

	var cursor *string
	if input.Cursor != "" {
		cursor = &input.Cursor
	}

	result, err := h.svc.ListMessages(ctx, input.MatchID, input.UserID, since, cursor, input.Limit)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	items := make([]matchMessagePayload, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, matchMessagePayload{
			ID:        item.ID,
			SenderID:  item.SenderID,
			Body:      item.Body,
			CreatedAt: item.CreatedAt.Format(time.RFC3339Nano),
		})
	}

	resp := &listMatchMessagesResponse{}
	resp.Body.Items = items
	resp.Body.NextCursor = result.NextCursor
	return resp, nil
}

func (h *messagingHandler) sendMessage(ctx context.Context, input *sendMatchMessageInput) (*sendMatchMessageResponse, error) {
	result, err := h.svc.SendMessage(ctx, input.MatchID, input.UserID, input.Body.Body)
	if err != nil {
		return nil, toHumaError(ctx, err)
	}

	resp := &sendMatchMessageResponse{}
	resp.Body = matchMessagePayload{
		ID:        result.ID,
		SenderID:  result.SenderID,
		Body:      result.Body,
		CreatedAt: result.CreatedAt.Format(time.RFC3339Nano),
	}
	return resp, nil
}

func (h *messagingHandler) closeMatch(ctx context.Context, input *closeMatchInput) (*closeMatchResponse, error) {
	if err := h.svc.CloseMatch(ctx, input.MatchID, input.UserID); err != nil {
		return nil, toHumaError(ctx, err)
	}

	resp := &closeMatchResponse{}
	resp.Body.Ok = true
	return resp, nil
}
