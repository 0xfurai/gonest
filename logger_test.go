package gonest

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestDefaultLogger_Log(t *testing.T) {
	var buf bytes.Buffer
	l := &DefaultLogger{logger: log.New(&buf, "[GoNest] ", 0)}

	l.Log("hello %s", "world")
	if !strings.Contains(buf.String(), "LOG   hello world") {
		t.Errorf("expected log output, got %q", buf.String())
	}
}

func TestDefaultLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	l := &DefaultLogger{logger: log.New(&buf, "", 0)}

	l.Error("something %s", "failed")
	if !strings.Contains(buf.String(), "ERROR something failed") {
		t.Errorf("expected error output, got %q", buf.String())
	}
}

func TestDefaultLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	l := &DefaultLogger{logger: log.New(&buf, "", 0)}

	l.Warn("warning %d", 42)
	if !strings.Contains(buf.String(), "WARN  warning 42") {
		t.Errorf("expected warn output, got %q", buf.String())
	}
}

func TestDefaultLogger_Debug_Disabled(t *testing.T) {
	var buf bytes.Buffer
	l := &DefaultLogger{logger: log.New(&buf, "", 0), debug: false}

	l.Debug("should not appear")
	if buf.Len() > 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestDefaultLogger_Debug_Enabled(t *testing.T) {
	var buf bytes.Buffer
	l := &DefaultLogger{logger: log.New(&buf, "", 0), debug: true}

	l.Debug("debug %s", "info")
	if !strings.Contains(buf.String(), "DEBUG debug info") {
		t.Errorf("expected debug output, got %q", buf.String())
	}
}

func TestNopLogger(t *testing.T) {
	l := NopLogger{}
	// Should not panic
	l.Log("test")
	l.Error("test")
	l.Warn("test")
	l.Debug("test")
}
