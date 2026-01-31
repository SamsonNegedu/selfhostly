package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/selfhostly/internal/gateway"
)

func main() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)
	logger := slog.Default()

	cfg, err := gateway.LoadConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("gateway configuration loaded",
		"primary_backend_url", cfg.PrimaryBackendURL,
		"listen_address", cfg.ListenAddress,
		"auth_enabled", cfg.AuthEnabled,
		"registry_ttl", cfg.RegistryTTL,
	)

	registry := gateway.NewNodeRegistry(cfg.PrimaryBackendURL, cfg.GatewayAPIKey, cfg.RegistryTTL, logger)
	registry.Start()

	router := gateway.NewRouter(registry, logger)
	proxy := gateway.NewProxy(router, registry, cfg, logger)

	server := &http.Server{
		Addr:         cfg.ListenAddress,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("gateway listening", "address", cfg.ListenAddress, "primary_backend", cfg.PrimaryBackendURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("gateway server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("gateway shutdown error", "error", err)
	}
	logger.Info("gateway stopped")
}
