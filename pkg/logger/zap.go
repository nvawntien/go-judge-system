package logger

import (
	"go-judge-system/pkg/config"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger(cfg config.LoggerConfig, mode string) *zap.Logger {
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})

	consoleWriter := zapcore.Lock(os.Stdout)

	encodeConfig := zap.NewProductionEncoderConfig()
	encodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if mode == "release" {
		encoder = zapcore.NewJSONEncoder(encodeConfig)
	} else {
		encodeConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encodeConfig)
	}

	level := zap.InfoLevel
	if cfg.Level == "debug" || mode == "debug" {
		level = zap.DebugLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, fileWriter, level),
		zapcore.NewCore(encoder, consoleWriter, level),
	)

	return zap.New(core, zap.AddCaller())
}
