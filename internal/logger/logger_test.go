package logger

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	logger := New()
	
	if logger == nil {
		t.Fatal("New() returned nil")
	}
	
	if logger.level != INFO {
		t.Errorf("Expected default level INFO, got %v", logger.level)
	}
	
	if logger.console == nil {
		t.Error("Console logger should be initialized")
	}
}

func TestSetLevel(t *testing.T) {
	logger := New()
	
	levels := []LogLevel{DEBUG, INFO, WARN, ERROR}
	
	for _, level := range levels {
		logger.SetLevel(level)
		if logger.level != level {
			t.Errorf("SetLevel(%v) failed, got %v", level, logger.level)
		}
	}
}

func TestSetLogFile(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	
	logger := New()
	
	if err := logger.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile() returned error: %v", err)
	}
	
	// Verify file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
	
	// Test that the file logger was set
	if logger.file == nil {
		t.Error("File handle should be set")
	}
	
	if logger.logger == nil {
		t.Error("File logger should be set")
	}
	
	// Clean up
	logger.Close()
}

func TestSetLogFileInvalidPath(t *testing.T) {
	logger := New()
	
	// Try to create a log file in a non-existent directory without permission
	invalidPath := "/invalid/path/test.log"
	
	err := logger.SetLogFile(invalidPath)
	if err == nil {
		t.Error("SetLogFile() should return error for invalid path")
	}
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	
	logger := New()
	
	// Test Close() when no file is set
	if err := logger.Close(); err != nil {
		t.Errorf("Close() with no file should not return error, got: %v", err)
	}
	
	// Set a log file
	if err := logger.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile() returned error: %v", err)
	}
	
	// Test Close() with file set
	if err := logger.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestLogLevels(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	
	logger := New()
	logger.SetLevel(DEBUG)
	
	if err := logger.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile() returned error: %v", err)
	}
	defer logger.Close()
	
	// Test all log levels
	logger.Debug("Debug message %d", 1)
	logger.Info("Info message %d", 2)
	logger.Warn("Warn message %d", 3)
	logger.Error("Error message %d", 4)
	
	// Read the log file content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	logContent := string(content)
	
	// Check that all messages were logged
	expectedMessages := []string{
		"DEBUG: Debug message 1",
		"INFO: Info message 2",
		"WARN: Warn message 3",
		"ERROR: Error message 4",
	}
	
	for _, expected := range expectedMessages {
		if !strings.Contains(logContent, expected) {
			t.Errorf("Log file should contain '%s', but got:\n%s", expected, logContent)
		}
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	
	logger := New()
	logger.SetLevel(WARN) // Only WARN and ERROR should be logged
	
	if err := logger.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile() returned error: %v", err)
	}
	defer logger.Close()
	
	// Test all log levels
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")
	
	// Read the log file content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	logContent := string(content)
	
	// Check that only WARN and ERROR messages were logged
	if strings.Contains(logContent, "DEBUG: Debug message") {
		t.Error("Debug message should not be logged when level is WARN")
	}
	
	if strings.Contains(logContent, "INFO: Info message") {
		t.Error("Info message should not be logged when level is WARN")
	}
	
	if !strings.Contains(logContent, "WARN: Warn message") {
		t.Error("Warn message should be logged when level is WARN")
	}
	
	if !strings.Contains(logContent, "ERROR: Error message") {
		t.Error("Error message should be logged when level is WARN")
	}
}

func TestConsoleOutput(t *testing.T) {
	// Capture console output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	logger := New()
	logger.SetLevel(DEBUG)
	
	// Only WARN and ERROR should go to console
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")
	
	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	consoleOutput := buf.String()
	
	// Check that only WARN and ERROR messages are in console output
	if strings.Contains(consoleOutput, "DEBUG: Debug message") {
		t.Error("Debug message should not be printed to console")
	}
	
	if strings.Contains(consoleOutput, "INFO: Info message") {
		t.Error("Info message should not be printed to console")
	}
	
	if !strings.Contains(consoleOutput, "WARN: Warn message") {
		t.Error("Warn message should be printed to console")
	}
	
	if !strings.Contains(consoleOutput, "ERROR: Error message") {
		t.Error("Error message should be printed to console")
	}
}

func TestLogMessageFormatting(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	
	logger := New()
	logger.SetLevel(DEBUG)
	
	if err := logger.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile() returned error: %v", err)
	}
	defer logger.Close()
	
	// Test message with formatting
	logger.Info("User %s performed action %d at %s", "john", 42, "2024-01-01")
	
	// Read the log file content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	logContent := string(content)
	
	// Check that the message was properly formatted
	expectedMessage := "User john performed action 42 at 2024-01-01"
	if !strings.Contains(logContent, expectedMessage) {
		t.Errorf("Log should contain formatted message '%s', but got:\n%s", expectedMessage, logContent)
	}
	
	// Check that timestamp and level are present
	if !strings.Contains(logContent, "INFO:") {
		t.Error("Log should contain INFO level")
	}
	
	// Check timestamp format (basic check for bracket format)
	if !strings.Contains(logContent, "[") || !strings.Contains(logContent, "]") {
		t.Error("Log should contain timestamp in brackets")
	}
}

func TestLogWithoutFile(t *testing.T) {
	// Capture console output for WARN/ERROR messages
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	logger := New()
	logger.SetLevel(DEBUG)
	
	// Log messages without setting a file
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")
	
	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	consoleOutput := buf.String()
	
	// Only WARN and ERROR should appear in console when no file is set
	if !strings.Contains(consoleOutput, "WARN: Warn message") {
		t.Error("Warn message should be printed to console when no file is set")
	}
	
	if !strings.Contains(consoleOutput, "ERROR: Error message") {
		t.Error("Error message should be printed to console when no file is set")
	}
	
	// DEBUG and INFO should not appear in console
	if strings.Contains(consoleOutput, "DEBUG: Debug message") {
		t.Error("Debug message should not be printed to console when no file is set")
	}
	
	if strings.Contains(consoleOutput, "INFO: Info message") {
		t.Error("Info message should not be printed to console when no file is set")
	}
}

func TestReplaceLogFile(t *testing.T) {
	tempDir := t.TempDir()
	logFile1 := filepath.Join(tempDir, "test1.log")
	logFile2 := filepath.Join(tempDir, "test2.log")
	
	logger := New()
	
	// Set first log file
	if err := logger.SetLogFile(logFile1); err != nil {
		t.Fatalf("SetLogFile(1) returned error: %v", err)
	}
	
	logger.Info("Message to file 1")
	
	// Set second log file (should close first file)
	if err := logger.SetLogFile(logFile2); err != nil {
		t.Fatalf("SetLogFile(2) returned error: %v", err)
	}
	
	logger.Info("Message to file 2")
	
	logger.Close()
	
	// Check first file
	content1, err := os.ReadFile(logFile1)
	if err != nil {
		t.Fatalf("Failed to read first log file: %v", err)
	}
	
	if !strings.Contains(string(content1), "Message to file 1") {
		t.Error("First log file should contain message 1")
	}
	
	if strings.Contains(string(content1), "Message to file 2") {
		t.Error("First log file should not contain message 2")
	}
	
	// Check second file
	content2, err := os.ReadFile(logFile2)
	if err != nil {
		t.Fatalf("Failed to read second log file: %v", err)
	}
	
	if strings.Contains(string(content2), "Message to file 1") {
		t.Error("Second log file should not contain message 1")
	}
	
	if !strings.Contains(string(content2), "Message to file 2") {
		t.Error("Second log file should contain message 2")
	}
}

func TestLogLevelConstants(t *testing.T) {
	// Test that log level constants have expected values
	if DEBUG != 0 {
		t.Errorf("Expected DEBUG = 0, got %d", DEBUG)
	}
	
	if INFO != 1 {
		t.Errorf("Expected INFO = 1, got %d", INFO)
	}
	
	if WARN != 2 {
		t.Errorf("Expected WARN = 2, got %d", WARN)
	}
	
	if ERROR != 3 {
		t.Errorf("Expected ERROR = 3, got %d", ERROR)
	}
}

func TestLoggerConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "concurrent.log")
	
	logger := New()
	logger.SetLevel(DEBUG)
	
	if err := logger.SetLogFile(logFile); err != nil {
		t.Fatalf("SetLogFile() returned error: %v", err)
	}
	defer logger.Close()
	
	// Test concurrent logging (basic test - not comprehensive)
	done := make(chan bool)
	
	go func() {
		for i := 0; i < 10; i++ {
			logger.Info("Goroutine 1 message %d", i)
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 10; i++ {
			logger.Error("Goroutine 2 message %d", i)
		}
		done <- true
	}()
	
	// Wait for both goroutines
	<-done
	<-done
	
	// Read the log file content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	
	logContent := string(content)
	
	// Check that messages from both goroutines are present
	// Note: This is a basic test and doesn't guarantee thread safety
	if !strings.Contains(logContent, "Goroutine 1") {
		t.Error("Log should contain messages from goroutine 1")
	}
	
	if !strings.Contains(logContent, "Goroutine 2") {
		t.Error("Log should contain messages from goroutine 2")
	}
}