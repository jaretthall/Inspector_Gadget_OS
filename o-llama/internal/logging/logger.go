package logging

import (
    "os"
    "strings"
    "sync"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var (
    logger     *zap.Logger
    once       sync.Once
    service    = "integrated-server"
    appVersion = ""
)

// Init sets global logger with the provided service and version values.
func Init(serviceName, version string) {
    service = serviceName
    appVersion = version
    once = sync.Once{}
    logger = nil
}

// L returns a global zap.SugaredLogger with default fields.
func L() *zap.SugaredLogger {
    once.Do(func() {
        logger = newZapLogger()
    })
    return logger.Sugar().With(
        "service", service,
        "version", appVersion,
        "env", getenvDefault("ENV", "dev"),
    )
}

func newZapLogger() *zap.Logger {
    level := parseLevel(getenvDefault("LOG_LEVEL", "info"))
    format := strings.ToLower(getenvDefault("LOG_FORMAT", "json"))

    encCfg := zapcore.EncoderConfig{
        TimeKey:        "ts",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        MessageKey:     "message",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.MillisDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }

    var encoder zapcore.Encoder
    if format == "console" {
        encoder = zapcore.NewConsoleEncoder(encCfg)
    } else {
        encoder = zapcore.NewJSONEncoder(encCfg)
    }

    core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
    opts := []zap.Option{zap.AddCaller(), zap.AddCallerSkip(1)}
    if getenvDefault("ENV", "dev") == "dev" {
        opts = append(opts, zap.Development())
    }
    return zap.New(core, opts...)
}

func parseLevel(s string) zapcore.Level {
    switch strings.ToLower(s) {
    case "trace":
        return zapcore.DebugLevel // Zap has no trace; map to debug
    case "debug":
        return zapcore.DebugLevel
    case "info":
        return zapcore.InfoLevel
    case "warn", "warning":
        return zapcore.WarnLevel
    case "error":
        return zapcore.ErrorLevel
    case "fatal":
        return zapcore.FatalLevel
    default:
        return zapcore.InfoLevel
    }
}

func getenvDefault(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}


