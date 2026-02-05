package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a production-grade logger with appropriate configuration
func NewLogger(isDevelopment bool) (*zap.Logger, error) {
	var config zap.Config

	if isDevelopment {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Allow DEBUG level in development
	if isDevelopment {
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewLoggerFromEnv creates logger based on environment
func NewLoggerFromEnv() (*zap.Logger, error) {
	env := os.Getenv("ENVIRONMENT")
	isDev := env == "" || env == "development" || env == "dev"
	return NewLogger(isDev)
}
