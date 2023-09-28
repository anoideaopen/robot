package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
)

var logger *zap.Logger

// GetLogger returns a logger
func GetLogger() *zap.Logger {
	if logger == nil {
		lvl, ok := os.LookupEnv("LOG_LEVEL")
		// LOG_LEVEL not set, let's default to debug
		if !ok {
			lvl = "error"
		}
		ll, err := zap.ParseAtomicLevel(lvl)
		if err != nil {
			ll = zap.NewAtomicLevelAt(zap.DebugLevel)
		}
		cfg := zap.NewProductionConfig()
		cfg.Level = ll
		logger, err = cfg.Build()
		if err != nil {
			panic(err)
		}
		defer logger.Sync()

		if err != nil {
			log.Fatalf("can't initialize zap logger: %v", err)
		}
		defer logger.Sync()
	}
	return logger
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	logger = GetLogger()
	logger.Error(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	logger = GetLogger()
	logger.Info(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	logger = GetLogger()
	logger.Debug(msg, fields...)
}
