// Package logger initialises a production-grade zap logger that writes
// structured JSON logs to a rotating file (ready for Fluent Bit ingestion)
// and a human-friendly console output in development mode.
package logger

import (
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Options controls logger initialisation.
type Options struct {
	// LogDir is the directory where app.log is written.
	LogDir string
	// MaxSizeMB is the maximum megabytes before the log file is rotated.
	MaxSizeMB int
	// MaxBackups is the maximum number of old log files to keep.
	MaxBackups int
	// MaxAgeDays is the maximum age in days before a log file is deleted.
	MaxAgeDays int
	// Development enables verbose console output (debug level, coloured keys).
	Development bool
}

var (
	global *zap.Logger
	once   sync.Once
)

// Init initialises the package-level logger. It is safe to call once from
// main; subsequent calls are no-ops (sync.Once).
func Init(opts Options) {
	once.Do(func() {
		global = build(opts)
		// Replace the global zap logger so third-party packages benefit too.
		zap.ReplaceGlobals(global)
	})
}

// L returns the package-level zap logger.
// Panics if Init has not been called yet.
func L() *zap.Logger {
	if global == nil {
		panic("logger: Init() must be called before L()")
	}
	return global
}

// S returns the package-level sugared logger (printf-style helpers).
func S() *zap.SugaredLogger { return L().Sugar() }

// Sync flushes any buffered log entries. Should be deferred in main.
func Sync() { _ = L().Sync() }

// ─── Internal builder ────────────────────────────────────────────────────────

func build(opts Options) *zap.Logger {
	// Ensure log directory exists.
	if err := os.MkdirAll(opts.LogDir, 0o755); err != nil {
		panic("logger: cannot create log directory: " + err.Error())
	}

	// File sink — JSON, rotated by lumberjack (Fluent Bit reads this file).
	fileSink := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(opts.LogDir, "app.log"),
		MaxSize:    opts.MaxSizeMB,
		MaxBackups: opts.MaxBackups,
		MaxAge:     opts.MaxAgeDays,
		Compress:   true,
	})

	// Console sink — plain stderr.
	consoleSink := zapcore.AddSync(os.Stderr)

	// Encoder configs.
	jsonEncCfg := zap.NewProductionEncoderConfig()
	jsonEncCfg.TimeKey = "timestamp"
	jsonEncCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleEncCfg := zap.NewDevelopmentEncoderConfig()
	consoleEncCfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	consoleEncCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	fileEncoder := zapcore.NewJSONEncoder(jsonEncCfg)
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncCfg)

	// Level: debug in development, info in production.
	level := zap.InfoLevel
	if opts.Development {
		level = zap.DebugLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileSink, level),
		zapcore.NewCore(consoleEncoder, consoleSink, level),
	)

	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	)

	return logger
}
