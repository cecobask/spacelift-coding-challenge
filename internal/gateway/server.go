package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

const (
	port = 3000
)

func StartServer(ctx context.Context, handler http.Handler) error {
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	return server.ListenAndServe()
}
