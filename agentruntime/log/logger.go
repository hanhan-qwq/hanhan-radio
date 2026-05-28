package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sugared *zap.SugaredLogger

func init() {
	format := os.Getenv("LOG_FORMAT")
	level := os.Getenv("LOG_LEVEL")

	var cfg zap.Config
	if format == "json" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
		// CLI tool — write logs to file, keep terminal clean.
		logFile := os.Getenv("LOG_FILE")
		if logFile == "" {
			logFile = "data/hanhan.log"
		}
		cfg.OutputPaths = []string{logFile}
		cfg.ErrorOutputPaths = []string{logFile}
	}

	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		// defaults: info for json, debug for console
	}

	logger, err := cfg.Build()
	if err != nil {
		panic("init zap logger: " + err.Error())
	}

	sugared = logger.Sugar()
}

// L returns the global sugared logger.
func L() *zap.SugaredLogger {
	return sugared
}

// Sync flushes any buffered log entries.
func Sync() {
	_ = sugared.Sync()
}
