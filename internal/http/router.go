package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Router builds the HTTP routes.
func Router() http.Handler {
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Amiglot API", "1.0.0"))

	huma.Get(api, "/healthz", func(ctx context.Context, input *struct{}) (*struct {
		Ok bool `json:"ok"`
	}, error) {
		return &struct {
			Ok bool `json:"ok"`
		}{Ok: true}, nil
	})

	return mux
}
