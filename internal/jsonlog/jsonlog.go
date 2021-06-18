package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

type Level int8

const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

// String ...
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Logger holds the output destination that the log entries
// will be written to, the minimum severity level that log
// entries will be written for, and a mutex for writes.
type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

// New create a new Logger
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

func (l *Logger) PrintInfo(msg string, props map[string]string) {
	l.print(LevelInfo, msg, props)
}

func (l *Logger) PrintErr(err error, props map[string]string) {
	l.print(LevelError, err.Error(), props)
}

func (l *Logger) PrintFatal(err error, props map[string]string) {
	l.print(LevelFatal, err.Error(), props)
	os.Exit(1)
}

func (l *Logger) print(level Level, msg string, props map[string]string) (int, error) {
	if level < l.minLevel {
		return 0, nil
	}
	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    msg,
		Properties: props,
	}

	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}
	var line []byte
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message:" + err.Error())
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out.Write(append(line, '\n'))
}

func (l *Logger) Write(msg []byte) (int, error) {
	return l.print(LevelError, string(msg), nil)
}
