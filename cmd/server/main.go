package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"xianhu-chaos/internal/chaos"
	"xianhu-chaos/internal/config"
	"xianhu-chaos/internal/httpserver"
	"xianhu-chaos/internal/logging"
	"xianhu-chaos/internal/provider"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	logger := slog.New(logging.NewColorHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}
	registry, err := provider.LoadRegistry(cfg.ProvidersDir)
	if err != nil {
		slog.Error("load providers failed", "error", err)
		os.Exit(1)
	}
	engine := chaos.New(registry, cfg.RequestLogLimit)
	server := httpserver.New(engine)

	slog.Info("xianhu-chaos listening", "addr", cfg.Server.Addr)
	if err := http.ListenAndServe(cfg.Server.Addr, server.Handler()); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
