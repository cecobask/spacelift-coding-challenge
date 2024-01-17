package gateway

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"
	"net/http"
)

func NewRouter(h *Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(
		slogchi.New(h.logger.WithGroup("http")),
		middleware.Recoverer,
	)
	router.Route("/object/{id}", func(r chi.Router) {
		r.Use(validateObjectID)
		r.Method(http.MethodGet, "/", handlerFunc(h.Get))
		r.Method(http.MethodPut, "/", handlerFunc(h.CreateOrUpdate))
	})
	return router
}
