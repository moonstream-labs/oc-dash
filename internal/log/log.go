// Package log provides a file-based structured logger for oc-dash.
//
// Since oc-dash runs as a full-screen TUI, stdout/stderr are owned by
// bubbletea. This logger writes to a file that can be tailed in a separate
// terminal for real-time troubleshooting:
//
//	tail -f ~/.local/share/oc-dash/oc-dash.log
package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level represents a log severity level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DBG"
	case LevelInfo:
		return "INF"
	case LevelWarn:
		return "WRN"
	case LevelError:
		return "ERR"
	default:
		return "???"
	}
}

// Logger is a concurrency-safe structured logger that writes to a file.
type Logger struct {
	mu       sync.Mutex
	w        io.Writer
	minLevel Level
	fields   []string // pre-formatted "key=value" pairs
}

var (
	global     *Logger
	globalOnce sync.Once
)

// DefaultLogPath returns the default log file path.
func DefaultLogPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "oc-dash", "oc-dash.log")
}

// Init initialises the global logger. If path is empty, logging is disabled
// (writes to io.Discard). Call this once from main.
func Init(path string, level Level) error {
	var w io.Writer = io.Discard

	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("log: mkdir: %w", err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("log: open: %w", err)
		}
		w = f
	}

	globalOnce.Do(func() {})
	global = &Logger{w: w, minLevel: level}

	// Write a separator so new runs are easy to spot in the log file.
	global.log(LevelInfo, "oc-dash started", nil)
	return nil
}

// Get returns the global logger (safe to call before Init — returns a no-op logger).
func Get() *Logger {
	if global == nil {
		return &Logger{w: io.Discard, minLevel: LevelError + 1}
	}
	return global
}

// With returns a child logger with additional key=value context fields.
func (l *Logger) With(kvs ...string) *Logger {
	if len(kvs)%2 != 0 {
		kvs = append(kvs, "MISSING")
	}
	fields := make([]string, len(l.fields), len(l.fields)+len(kvs)/2)
	copy(fields, l.fields)
	for i := 0; i < len(kvs); i += 2 {
		fields = append(fields, kvs[i]+"="+kvs[i+1])
	}
	return &Logger{w: l.w, minLevel: l.minLevel, fields: fields}
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, kvs ...string) { l.log(LevelDebug, msg, kvs) }

// Info logs at info level.
func (l *Logger) Info(msg string, kvs ...string) { l.log(LevelInfo, msg, kvs) }

// Warn logs at warn level.
func (l *Logger) Warn(msg string, kvs ...string) { l.log(LevelWarn, msg, kvs) }

// Error logs at error level.
func (l *Logger) Error(msg string, kvs ...string) { l.log(LevelError, msg, kvs) }

// Err is a convenience for logging an error value.
func (l *Logger) Err(msg string, err error, kvs ...string) {
	kvs = append(kvs, "err", err.Error())
	l.log(LevelError, msg, kvs)
}

func (l *Logger) log(level Level, msg string, kvs []string) {
	if level < l.minLevel {
		return
	}

	var sb strings.Builder
	sb.WriteString(time.Now().Format("15:04:05.000"))
	sb.WriteByte(' ')
	sb.WriteString(level.String())
	sb.WriteByte(' ')
	sb.WriteString(msg)

	// Append pre-set fields
	for _, f := range l.fields {
		sb.WriteByte(' ')
		sb.WriteString(f)
	}
	// Append inline fields
	if len(kvs)%2 != 0 {
		kvs = append(kvs, "MISSING")
	}
	for i := 0; i < len(kvs); i += 2 {
		sb.WriteByte(' ')
		sb.WriteString(kvs[i])
		sb.WriteByte('=')
		sb.WriteString(kvs[i+1])
	}
	sb.WriteByte('\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.w.Write([]byte(sb.String()))
}
