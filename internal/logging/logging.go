// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package logging

import (
	"log"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Environment variable to control log level
	EnvLogLevel = "TF_LOG"
	// Environment variable to control log format
	EnvLogFormat = "TF_LOG_FORMAT"
)

var (
	// Valid log levels
	ValidLevels = []string{"DEBUG", "INFO", "WARN", "ERROR", "OFF"}
	// Valid log formats
	ValidFormats = []string{"JSON", "CONSOLE"}
	// Global logger instance
	logger *zap.Logger
	// Sugar logger for convenience methods
	sugar *zap.SugaredLogger
)

// LoggerOptions holds configuration for the logger
type LoggerOptions struct {
	PlatformType string
}

// parseLogLevel converts string level to zapcore.Level
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARN":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	case "OFF":
		return zapcore.FatalLevel + 1 // Higher than any valid level
	default:
		return zapcore.InfoLevel // Default to INFO
	}
}

// SetupLogger initializes the global logger
func SetupLogger(options *LoggerOptions) {
	if options == nil {
		options = &LoggerOptions{}
	}

	// Read log level from env var
	logLevelStr := os.Getenv(EnvLogLevel)
	if logLevelStr == "" {
		logLevelStr = "INFO" // Default level
	}
	logLevel := parseLogLevel(logLevelStr)

	// Read log format from env var
	logFormat := os.Getenv(EnvLogFormat)
	if logFormat == "" {
		logFormat = "CONSOLE" // Default format
	}
	logFormat = strings.ToUpper(logFormat)

	// Configure encoder based on format
	var encoder zapcore.Encoder
	if logFormat == "JSON" {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
		encoderConfig.ConsoleSeparator = " "
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stderr),
		logLevel,
	)

	// Create logger with platform field
	logger = zap.New(core, 
		zap.AddCaller(), 
		zap.AddCallerSkip(1),
		zap.Fields(zap.String("platform", options.PlatformType)),
	)

	// Create sugared logger for convenience methods
	sugar = logger.Sugar()

	// Redirect standard library's logger to zap
	zap.RedirectStdLog(logger)

	// Log initialization
	if logLevel <= zapcore.DebugLevel {
		sugar.Debugw("Logger initialized",
			"level", logLevelStr,
			"format", logFormat,
			"platform", options.PlatformType,
		)
	}
}

// GetLogger returns the Zap logger
func GetLogger() *zap.Logger {
	return logger
}

// GetSugaredLogger returns the sugared Zap logger
func GetSugaredLogger() *zap.SugaredLogger {
	return sugar
}

// Debug logs a message at debug level
func Debug(msg string, args ...interface{}) {
	if logger == nil {
		log.Printf("[DEBUG] %s", msg)
		return
	}
	
	if len(args) == 0 {
		sugar.Debug(msg)
	} else if len(args)%2 == 0 {
		sugar.Debugw(msg, args...)
	} else {
		sugar.Debugf(msg, args...)
	}
}

// Info logs a message at info level
func Info(msg string, args ...interface{}) {
	if logger == nil {
		log.Printf("[INFO] %s", msg)
		return
	}
	
	if len(args) == 0 {
		sugar.Info(msg)
	} else if len(args)%2 == 0 {
		sugar.Infow(msg, args...)
	} else {
		sugar.Infof(msg, args...)
	}
}

// Warn logs a message at warn level
func Warn(msg string, args ...interface{}) {
	if logger == nil {
		log.Printf("[WARN] %s", msg)
		return
	}
	
	if len(args) == 0 {
		sugar.Warn(msg)
	} else if len(args)%2 == 0 {
		sugar.Warnw(msg, args...)
	} else {
		sugar.Warnf(msg, args...)
	}
}

// Error logs a message at error level
func Error(msg string, args ...interface{}) {
	if logger == nil {
		log.Printf("[ERROR] %s", msg)
		return
	}
	
	if len(args) == 0 {
		sugar.Error(msg)
	} else if len(args)%2 == 0 {
		sugar.Errorw(msg, args...)
	} else {
		sugar.Errorf(msg, args...)
	}
}

// Sync flushes any buffered log entries
func Sync() error {
	if logger == nil {
		return nil
	}
	return logger.Sync()
}