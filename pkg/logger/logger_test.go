package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, INFO)

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO in output, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
}

func TestLogger_Infof(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, INFO)

	logger.Infof("test %s %d", "message", 42)

	output := buf.String()
	if !strings.Contains(output, "test message 42") {
		t.Errorf("Expected 'test message 42' in output, got: %s", output)
	}
}

func TestLogger_Warning(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, WARNING)

	logger.Warning("warning message")

	output := buf.String()
	if !strings.Contains(output, "WARNING") {
		t.Errorf("Expected WARNING in output, got: %s", output)
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	// ERROR vai para stderr, então passamos buf como stderr também
	logger := &Logger{
		errorLogger: log.New(&buf, "ERROR:   ", log.Ldate|log.Ltime|log.Lmicroseconds),
		minLevel:    ERROR,
	}

	logger.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected ERROR in output, got: %s", output)
	}
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected 'error message' in output, got: %s", output)
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Testa que as funções package-level não causam panic
	Info("test info")
	Infof("test %s", "infof")
	Warning("test warning")
	Warningf("test %s", "warningf")
	Error("test error")
	Errorf("test %s", "errorf")

	// Se chegou aqui sem panic, passou
}
