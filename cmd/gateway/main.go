package main

import (
	"context"
	"github.com/cecobask/spacelift-coding-challenge/internal/container"
	"github.com/cecobask/spacelift-coding-challenge/internal/gateway"
	"github.com/cecobask/spacelift-coding-challenge/internal/storage"
	"github.com/cecobask/spacelift-coding-challenge/pkg/log"
)

func main() {
	logger := log.DefaultLogger()
	logger.Info("starting gateway")
	ctx := log.WithContext(context.Background(), logger)
	docker, err := container.NewDocker()
	logger.ExitOnError(err)
	minio := storage.NewMinio(docker)
	err = minio.Setup(ctx)
	logger.ExitOnError(err)
	handler := gateway.NewHandler(minio)
	router := gateway.NewRouter(handler)
	logger.Info("starting http server")
	err = gateway.StartServer(ctx, router)
	logger.ExitOnError(err)
}
