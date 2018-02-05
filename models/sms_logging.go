package models

import (
	"go.uber.org/zap"
	"os"
	"go.uber.org/zap/zapcore"
)

var unsetEnvVar = ""
var logr *zap.Logger
var sugaredLogr *zap.SugaredLogger

func Logger() *zap.Logger {
	if logr == nil {
		if isProduction() {
			encoderConfig := zapcore.EncoderConfig{
				// Keys can be anything except the empty string.
				TimeKey:        "T",
				MessageKey:     "M",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.StringDurationEncoder,
			}
			config := zap.Config{
				Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
				Development:      false,
				Encoding:         "console",
				EncoderConfig:    encoderConfig,
				OutputPaths:      []string{"stdout"},
				ErrorOutputPaths: []string{"stderr"},
			}
			logr, _ = config.Build()
		} else {
			logr, _ = zap.NewDevelopment()
		}
	}

	return logr
}

func isProduction() bool {
	return os.Getenv("DEBUG") == unsetEnvVar
}

func SugaredLogger() *zap.SugaredLogger {
	if sugaredLogr == nil {
		sugaredLogr = Logger().Sugar()
	}

	return sugaredLogr
}
