package main

import (
	"context"
	"os"

	"go.uber.org/zap"

	"github.com/wyw14/cry045/internal/application"
	"github.com/wyw14/cry045/internal/config"
	"github.com/wyw14/cry045/internal/repository"
	httptransport "github.com/wyw14/cry045/internal/transport/http"
)

func main() {
	cfg := config.Load()
	log, _ := zap.NewProduction()
	defer log.Sync()
	store := repository.NewDemoStore(application.RealClock{}.Now())
	service := application.NewComplianceService(store, application.RealClock{})
	server := httptransport.NewServer(service, log)
	if err := server.Engine.Run(cfg.HTTPAddr); err != nil {
		log.Error("server stopped", zap.Error(err))
		os.Exit(1)
	}
	_ = context.Background()
}
