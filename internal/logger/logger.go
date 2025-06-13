package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level   LogLevel
	file    *os.File
	logger  *log.Logger
	console *log.Logger
}

func New() *Logger {
	return &Logger{
		level:   INFO,
		console: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) SetLogFile(filename string) error {
	if l.file != nil {
		l.file.Close()
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.file = file
	l.logger = log.New(file, "", 0)
	return nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) log(level LogLevel, format string, args ...any) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := []string{"DEBUG", "INFO", "WARN", "ERROR"}[level]
	message := fmt.Sprintf(format, args...)

	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, levelStr, message)

	if l.logger != nil {
		l.logger.Println(logLine)
	}

	if level >= WARN {
		l.console.Println(logLine)
	}
}

func (l *Logger) Debug(format string, args ...any) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...any) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...any) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.log(ERROR, format, args...)
}
