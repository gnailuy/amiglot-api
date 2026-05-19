package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/buildinfo"
	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/i18n"
)

// Router builds the HTTP routes.
func Router(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	root := http.NewServeMux()
	apiMux := http.NewServeMux()
	api := humago.New(apiMux, huma.DefaultConfig("Amiglot API", "1.0.0"))

	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		locale := i18n.LocaleFromHeader(ctx.Header("Accept-Language"))
		ctx = huma.WithContext(ctx, i18n.ContextWithLocale(ctx.Context(), locale))
		next(ctx)
	})

	huma.Get(api, "/healthz", func(ctx context.Context, input *struct{}) (*struct {
		Body struct {
			Ok           bool   `json:"ok"`
			GitSHA       string `json:"git_sha"`
			GitBranch    string `json:"git_branch"`
			BuildTimeUTC string `json:"build_time_utc"`
		} `json:""`
	}, error) {
		return &struct {
			Body struct {
				Ok           bool   `json:"ok"`
				GitSHA       string `json:"git_sha"`
				GitBranch    string `json:"git_branch"`
				BuildTimeUTC string `json:"build_time_utc"`
			} `json:""`
		}{
			Body: struct {
				Ok           bool   `json:"ok"`
				GitSHA       string `json:"git_sha"`
				GitBranch    string `json:"git_branch"`
				BuildTimeUTC string `json:"build_time_utc"`
			}{
				Ok:           true,
				GitSHA:       buildinfo.GitSHA,
				GitBranch:    buildinfo.GitBranch,
				BuildTimeUTC: buildinfo.BuildTimeUTC,
			},
		}, nil
	})

	registerAuthRoutes(api, cfg, pool)
	registerProfileRoutes(api, pool)
	registerDiscoveryRoutes(api, cfg, pool)
	registerConnectionRoutes(api, cfg, pool)
	registerMessagingRoutes(api, cfg, pool)

	root.Handle("/api/v1/", http.StripPrefix("/api/v1", apiMux))

	return root
}
