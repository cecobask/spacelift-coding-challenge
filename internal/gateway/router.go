package gateway

import (
	"fmt"
	"github.com/cecobask/spacelift-coding-challenge/pkg/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
)

func NewRouter(h *Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(
		requestLogger,
		middleware.Recoverer,
	)
	router.Route("/object/{id}", func(r chi.Router) {
		r.Use(validateObjectID)
		r.Method(http.MethodGet, "/", handlerFunc(h.Get))
		r.Method(http.MethodPut, "/", handlerFunc(h.CreateOrUpdate))
	})
	return router
}

func validateObjectID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if len(id) > 32 {
			message := fmt.Sprintf("object id must be between 1-32 characters long, but received invalid value: %s", id)
			http.Error(w, message, http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attributes := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("host", r.Host),
		}
		logger := log.FromContext(r.Context())
		logger.WithGroup("http").LogAttrs(r.Context(), slog.LevelInfo, "received http request", attributes...)
		next.ServeHTTP(w, r)
	})
}
