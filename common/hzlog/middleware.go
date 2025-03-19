package hzlog

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func InjectRequestIDToLogContext() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			rID := middleware.GetReqID(ctx)
			lgCtx := ContextWith(ctx, slog.String("request_id", rID))

			h.ServeHTTP(w, r.WithContext(lgCtx))
		})
	}
}
