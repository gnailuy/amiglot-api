package http

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/config"
)

type authHandler struct {
	cfg  config.Config
	pool *pgxpool.Pool
}

func registerAuthRoutes(api huma.API, cfg config.Config, pool *pgxpool.Pool) {
	h := &authHandler{cfg: cfg, pool: pool}

	huma.Post(api, "/auth/magic-link", h.requestMagicLink)
	huma.Post(api, "/auth/verify", h.verifyMagicLink)
	huma.Post(api, "/auth/logout", h.logout)
}

type magicLinkRequest struct {
	Body struct {
		Email string `json:"email"`
	}
}

type magicLinkResponse struct {
	Ok          bool    `json:"ok"`
	DevLoginURL *string `json:"dev_login_url,omitempty"`
}

func (h *authHandler) requestMagicLink(ctx context.Context, input *magicLinkRequest) (*magicLinkResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}

	email := strings.TrimSpace(strings.ToLower(input.Body.Email))
	if email == "" {
		return nil, huma.Error400BadRequest("email is required")
	}

	userID, err := h.ensureUser(ctx, email)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to load user")
	}

	token, tokenHash, err := generateToken()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to generate token")
	}

	expiresAt := time.Now().Add(h.cfg.MagicLinkTTL)
	if _, err := h.pool.Exec(ctx,
		`INSERT INTO magic_link_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID,
		tokenHash,
		expiresAt,
	); err != nil {
		return nil, huma.Error500InternalServerError("failed to store token")
	}

	var devLoginURL *string
	if h.cfg.Env == "dev" {
		link := h.cfg.MagicLinkBaseURL + "?token=" + token
		devLoginURL = &link
		log.Printf("dev magic link for %s: %s", email, link)
	} else {
		log.Printf("magic link requested for %s", email)
	}

	return &magicLinkResponse{Ok: true, DevLoginURL: devLoginURL}, nil
}

type verifyRequest struct {
	Body struct {
		Token string `json:"token"`
	}
}

type verifyResponse struct {
	AccessToken string `json:"access_token"`
	User        struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (h *authHandler) verifyMagicLink(ctx context.Context, input *verifyRequest) (*verifyResponse, error) {
	if h.pool == nil {
		return nil, huma.Error503ServiceUnavailable("database unavailable")
	}

	token := strings.TrimSpace(input.Body.Token)
	if token == "" {
		return nil, huma.Error400BadRequest("token is required")
	}

	tokenHash := sha256.Sum256([]byte(token))

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start transaction")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var tokenID string
	var userID string
	row := tx.QueryRow(ctx,
		`SELECT id, user_id FROM magic_link_tokens
		 WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
		 FOR UPDATE`,
		tokenHash[:],
	)
	if err := row.Scan(&tokenID, &userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, huma.Error401Unauthorized("invalid or expired token")
		}
		return nil, huma.Error500InternalServerError("failed to load token")
	}

	if _, err := tx.Exec(ctx, `UPDATE magic_link_tokens SET consumed_at = now() WHERE id = $1`, tokenID); err != nil {
		return nil, huma.Error500InternalServerError("failed to consume token")
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to update login time")
	}

	var email string
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		return nil, huma.Error500InternalServerError("failed to load user")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, huma.Error500InternalServerError("failed to commit token")
	}

	accessToken, _, err := generateToken()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to generate access token")
	}

	resp := &verifyResponse{AccessToken: accessToken}
	resp.User.ID = userID
	resp.User.Email = email

	return resp, nil
}

type logoutResponse struct {
	Ok bool `json:"ok"`
}

func (h *authHandler) logout(ctx context.Context, input *struct{}) (*logoutResponse, error) {
	return &logoutResponse{Ok: true}, nil
}

func (h *authHandler) ensureUser(ctx context.Context, email string) (string, error) {
	var id string
	err := h.pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	err = h.pool.QueryRow(ctx, `INSERT INTO users (email) VALUES ($1) RETURNING id`, email).Scan(&id)
	if err != nil {
		return "", err
	}

	return id, nil
}

func generateToken() (string, []byte, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(encoded))
	return encoded, hash[:], nil
}
