package logger

import (
	"log/slog"
	"os"
)

// InitLogger initializes and configures the application logger based on environment
// Returns a configured slog.Logger instance
// useJSON: if true, use JSON handler; if false, use text handler
// environment: used to determine log level and source info (development enables debug level)
func InitLogger(environment string, useJSON bool) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// In development, use more verbose logging
	if environment == "development" {
		opts.Level = slog.LevelDebug
		opts.AddSource = true // Include source file and line number
	}

	// Choose handler based on useJSON parameter
	if useJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	
	// Set as default logger so it can be used throughout the application
	slog.SetDefault(logger)

	return logger
}
