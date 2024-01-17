package main

import (
	"context"
	"github.com/cecobask/spacelift-coding-challenge/internal/gateway"
	"github.com/cecobask/spacelift-coding-challenge/internal/storage"
	"github.com/cecobask/spacelift-coding-challenge/pkg/log"
	"net/http"
)

func main() {
	ctx := context.Background()
	logger := log.DefaultLogger()
	minio, err := storage.NewMinio(ctx, logger)
	logger.ExitOnError(err)
	handler := gateway.NewHandler(ctx, logger, minio)
	router := gateway.NewRouter(handler)
	logger.Info("Starting server on port 3000")
	err = http.ListenAndServe(":3000", router)
	logger.ExitOnError(err)
}
