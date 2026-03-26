package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{w: &buf, minLevel: LevelInfo}

	l.Debug("should not appear")
	l.Info("should appear")
	l.Warn("also appears")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("debug message should have been filtered")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("info message should be present")
	}
	if !strings.Contains(output, "also appears") {
		t.Error("warn message should be present")
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{w: &buf, minLevel: LevelDebug}

	child := l.With("pkg", "test", "component", "widget")
	child.Info("hello")

	output := buf.String()
	if !strings.Contains(output, "pkg=test") {
		t.Error("expected pkg=test in output")
	}
	if !strings.Contains(output, "component=widget") {
		t.Error("expected component=widget in output")
	}
	if !strings.Contains(output, "hello") {
		t.Error("expected message in output")
	}
}

func TestLoggerInlineKVs(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{w: &buf, minLevel: LevelDebug}

	l.Info("test", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected key=value in output, got: %s", output)
	}
}

func TestLoggerErr(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{w: &buf, minLevel: LevelDebug}

	l.Err("something broke", &testErr{msg: "connection refused"})

	output := buf.String()
	if !strings.Contains(output, "ERR") {
		t.Error("expected ERR level")
	}
	if !strings.Contains(output, "err=connection refused") {
		t.Errorf("expected err=connection refused, got: %s", output)
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DBG"},
		{LevelInfo, "INF"},
		{LevelWarn, "WRN"},
		{LevelError, "ERR"},
		{Level(99), "???"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestGetBeforeInit(t *testing.T) {
	// Reset global for test isolation
	saved := global
	global = nil
	defer func() { global = saved }()

	l := Get()
	// Should not panic, should be a no-op logger
	l.Info("this should not panic")
	l.Error("nor this")
}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
