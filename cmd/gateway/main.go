package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/selfhostly/internal/gateway"
	"github.com/selfhostly/internal/logger"
)

func main() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)
	
	// Load environment and logging config
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "production"
	}
	
	// Determine JSON logging preference (same logic as backend)
	logJSONEnv := os.Getenv("LOG_JSON")
	var logJSON bool
	if logJSONEnv != "" {
		logJSON = logJSONEnv == "true"
	} else {
		// Default: JSON in production, text in development
		logJSON = environment != "development"
	}
	
	// Initialize logger with configuration
	appLogger := logger.InitLogger(environment, logJSON)

	cfg, err := gateway.LoadConfig()
	if err != nil {
		appLogger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	appLogger.Info("gateway configuration loaded",
		"primary_backend_url", cfg.PrimaryBackendURL,
		"listen_address", cfg.ListenAddress,
		"auth_enabled", cfg.AuthEnabled,
		"registry_ttl", cfg.RegistryTTL,
	)

	registry := gateway.NewNodeRegistry(cfg.PrimaryBackendURL, cfg.GatewayAPIKey, cfg.RegistryTTL, appLogger)
	registry.Start()

	router := gateway.NewRouter(registry, appLogger)
	proxy := gateway.NewProxy(router, registry, cfg, appLogger)

	server := &http.Server{
		Addr:         cfg.ListenAddress,
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		appLogger.Info("gateway listening", "address", cfg.ListenAddress, "primary_backend", cfg.PrimaryBackendURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("gateway server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		appLogger.Error("gateway shutdown error", "error", err)
	}
	appLogger.Info("gateway stopped")
}
