// Package logx provides a small leveled logger with hierarchical indentation
// and optional per-build file output. It mirrors the behaviour of the original
// Python CustomLogger (indent levels + level prefixes) without pulling in a
// heavyweight logging framework.
package logx

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level is a log severity.
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
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARNING"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// ParseLevel converts a config string ("debug", "info", ...) to a Level.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

const indentSpaces = 4

// Logger writes to a console writer and, optionally, a file writer. It is safe
// for concurrent use. Indentation is global to the logger to match the original
// program's nested log style.
type Logger struct {
	mu      sync.Mutex
	min     Level
	colors  bool
	console io.Writer
	file    *os.File
	indent  int
}

// New creates a logger writing to stdout at the given minimum level.
func New(min Level, colors bool) *Logger {
	return &Logger{min: min, colors: colors, console: os.Stdout}
}

// SetFile directs log output to a file as well as the console. The file is
// truncated. The parent directory must already exist. Passing an empty path
// detaches any current file.
func (l *Logger) SetFile(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	l.file = f
	return nil
}

// Indent increases the indentation level for subsequent messages.
func (l *Logger) Indent() {
	l.mu.Lock()
	l.indent++
	l.mu.Unlock()
}

// Dedent decreases the indentation level, never going below zero.
func (l *Logger) Dedent() {
	l.mu.Lock()
	if l.indent > 0 {
		l.indent--
	}
	l.mu.Unlock()
}

func (l *Logger) log(level Level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.min {
		return
	}

	prefix := strings.Repeat(" ", l.indent*indentSpaces)
	levelPrefix := ""
	if level >= LevelWarn {
		levelPrefix = level.String() + ": "
	}
	line := prefix + levelPrefix + msg

	if l.console != nil {
		if l.colors {
			fmt.Fprintln(l.console, colorize(level, line))
		} else {
			fmt.Fprintln(l.console, line)
		}
	}
	if l.file != nil {
		fmt.Fprintln(l.file, line)
	}
}

func (l *Logger) Debug(format string, a ...any) { l.log(LevelDebug, fmt.Sprintf(format, a...)) }
func (l *Logger) Info(format string, a ...any)  { l.log(LevelInfo, fmt.Sprintf(format, a...)) }
func (l *Logger) Warn(format string, a ...any)  { l.log(LevelWarn, fmt.Sprintf(format, a...)) }
func (l *Logger) Error(format string, a ...any) { l.log(LevelError, fmt.Sprintf(format, a...)) }

// PrintTime logs the current local time, matching the original print_time().
func (l *Logger) PrintTime() {
	l.Info("%s", time.Now().Format("01-02-06 15:04:05"))
}

func colorize(level Level, s string) string {
	const reset = "\033[0m"
	var code string
	switch level {
	case LevelDebug:
		code = "\033[90m" // grey
	case LevelWarn:
		code = "\033[33m" // yellow
	case LevelError:
		code = "\033[31m" // red
	default:
		return s // INFO uncoloured
	}
	return code + s + reset
}
