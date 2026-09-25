package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ANSI colors — only emitted when the writer is a TTY.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// Logger wraps slog with a custom text handler and helper methods.
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration.
type Config struct {
	Level      string
	Format     string
	Output     string
	FilePath   string
	Service    string
	MaxSizeMB  int // rotate after this many MB (default 100)
	MaxBackups int // keep this many old files (default 5)
	MaxAgeDays int // delete files older than N days (default 30)
	Compress   bool
}

// New creates a new Logger.
func New(cfg *Config) *Logger {
	if cfg == nil {
		cfg = &Config{
			Level:   "info",
			Format:  "text",
			Output:  "stdout",
			Service: "analytics-server",
		}
	}

	level := parseLevel(cfg.Level)

	writer, err := createWriter(cfg)
	if err != nil {
		// Fall back to stdout — logging must never prevent startup.
		fmt.Fprintf(os.Stderr, "logger: falling back to stdout: %v\n", err)
		writer = os.Stdout
	}

	useColors := isTerminal(writer)

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	} else {
		handler = newTextHandler(writer, level, useColors)
	}

	handler = handler.WithAttrs([]slog.Attr{slog.String("service", cfg.Service)})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return &Logger{Logger: logger}
}

// ---------------------------------------------------------------------------
// TEXT HANDLER
// ---------------------------------------------------------------------------

// textHandler is a custom slog.Handler that prints colored, human-readable logs.
type textHandler struct {
	writer io.Writer
	level  slog.Level
	colors bool
	attrs  []slog.Attr
}

func newTextHandler(w io.Writer, level slog.Level, colors bool) *textHandler {
	return &textHandler{
		writer: w,
		level:  level,
		colors: colors,
		attrs:  []slog.Attr{},
	}
}

func (h *textHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *textHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder

	// Time
	if h.colors {
		b.WriteString(colorGray)
		b.WriteString(r.Time.Format("15:04:05"))
		b.WriteString(colorReset)
	} else {
		b.WriteString(r.Time.Format("2006-01-02T15:04:05.000Z07:00"))
	}
	b.WriteString(" ")

	// Level
	levelTag, levelColor := formatLevel(r.Level, h.colors)
	if h.colors {
		b.WriteString(levelColor)
		b.WriteString(levelTag)
		b.WriteString(colorReset)
	} else {
		b.WriteString(levelTag)
	}
	b.WriteString(" ")

	// Message — quote if it contains spaces
	b.WriteString(quoteIfNeeded(r.Message))

	// Handler-level attributes
	for _, attr := range h.attrs {
		if attr.Key != "service" {
			writeAttr(&b, attr, h.colors)
		}
	}

	// Record-level attributes
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "service" {
			writeAttr(&b, a, h.colors)
		}
		return true
	})

	b.WriteString("\n")

	_, err := fmt.Fprint(h.writer, b.String())
	return err
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)
	return &textHandler{
		writer: h.writer,
		level:  h.level,
		colors: h.colors,
		attrs:  newAttrs,
	}
}

func (h *textHandler) WithGroup(_ string) slog.Handler {
	return h
}

// ---------------------------------------------------------------------------
// HELPERS
// ---------------------------------------------------------------------------

func formatLevel(level slog.Level, withColor bool) (string, string) {
	switch {
	case level < slog.LevelInfo:
		return "DBG", colorCyan
	case level < slog.LevelWarn:
		return "INF", colorGreen
	case level < slog.LevelError:
		return "WRN", colorYellow
	default:
		return "ERR", colorRed
	}
}

func writeAttr(b *strings.Builder, attr slog.Attr, colors bool) {
	b.WriteString(" ")
	if colors {
		b.WriteString(colorGray)
		b.WriteString(attr.Key)
		b.WriteString("=")
		b.WriteString(colorReset)
	} else {
		b.WriteString(attr.Key)
		b.WriteString("=")
	}
	b.WriteString(quoteIfNeeded(attr.Value.String()))
}

// quoteIfNeeded wraps a string in quotes when it contains spaces or
// special characters. Makes the log easier to read and grep.
func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '=' {
			return fmt.Sprintf("%q", s)
		}
	}
	return s
}

// isTerminal returns true if the writer is a *os.File pointing at a TTY.
// Colors are only enabled in that case.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ---------------------------------------------------------------------------
// LOGGER METHODS
// ---------------------------------------------------------------------------

// WithRequestID returns a logger with the request_id attribute.
func (l *Logger) WithRequestID(requestID string) *Logger {
	if requestID == "" {
		return l
	}
	return &Logger{Logger: l.Logger.With("request_id", requestID)}
}

// WithError returns a logger with the error attribute.
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{Logger: l.Logger.With("error", err.Error())}
}

// With returns a logger with additional key-value pairs.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...)}
}

// ---------------------------------------------------------------------------
// CONFIG PARSING
// ---------------------------------------------------------------------------

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func createWriter(cfg *Config) (io.Writer, error) {
	if cfg.Output != "file" && cfg.Output != "both" {
		return os.Stdout, nil
	}

	path := cfg.FilePath
	if path == "" {
		path = "logs/app.log"
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir log dir: %w", err)
	}

	rotator := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    cfg.MaxSizeMB,  // MB
		MaxBackups: cfg.MaxBackups, // number of files
		MaxAge:     cfg.MaxAgeDays, // days
		Compress:   cfg.Compress,
	}

	if cfg.Output == "both" {
		return io.MultiWriter(os.Stdout, rotator), nil
	}
	return rotator, nil
}
