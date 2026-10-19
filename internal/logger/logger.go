// Package logger provides structured logging for the agent.
// It writes JSON-structured entries to both stderr and an optional log file.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Level represents log severity.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a Level.
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Entry is a single structured log record.
type Entry struct {
	Time    string      `json:"time"`
	Level   string      `json:"level"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
}

// Logger provides leveled, structured logging.
type Logger struct {
	mu      sync.Mutex
	level   Level
	writers []io.Writer
	file    *os.File
}

// New creates a Logger. If logFile is non-empty, logs are also written there.
func New(level string, logFile string) (*Logger, error) {
	l := &Logger{
		level:   ParseLevel(level),
		writers: []io.Writer{os.Stderr},
	}

	if logFile != "" {
		dir := filepath.Dir(logFile)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("logger: create log dir: %w", err)
		}
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("logger: open log file: %w", err)
		}
		l.file = f
		l.writers = append(l.writers, f)
	}

	return l, nil
}

// Close flushes and closes the log file, if any.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) log(lvl Level, msg string, data interface{}) {
	if lvl < l.level {
		return
	}
	entry := Entry{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   lvl.String(),
		Message: msg,
		Data:    data,
	}
	raw, _ := json.Marshal(entry)
	raw = append(raw, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.writers {
		_, _ = w.Write(raw)
	}
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(msg string, data ...interface{}) {
	var d interface{}
	if len(data) > 0 {
		d = data[0]
	}
	l.log(DEBUG, msg, d)
}

// Info logs at INFO level.
func (l *Logger) Info(msg string, data ...interface{}) {
	var d interface{}
	if len(data) > 0 {
		d = data[0]
	}
	l.log(INFO, msg, d)
}

// Warn logs at WARN level.
func (l *Logger) Warn(msg string, data ...interface{}) {
	var d interface{}
	if len(data) > 0 {
		d = data[0]
	}
	l.log(WARN, msg, d)
}

// Error logs at ERROR level.
func (l *Logger) Error(msg string, data ...interface{}) {
	var d interface{}
	if len(data) > 0 {
		d = data[0]
	}
	l.log(ERROR, msg, d)
}

// ActionLog records a tool action with its result for auditing.
type ActionLog struct {
	Tool      string      `json:"tool"`
	Server    string      `json:"server"`
	Arguments interface{} `json:"arguments,omitempty"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Duration  string      `json:"duration,omitempty"`
}

// LogAction writes a structured tool action entry.
func (l *Logger) LogAction(action ActionLog) {
	l.log(INFO, "tool_call", action)
}
