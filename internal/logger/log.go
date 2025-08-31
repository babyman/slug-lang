package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
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
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	level      Level
	color      bool
	toFile     bool
	fileHandle *os.File
	logger     *log.Logger
	mu         sync.Mutex
	prefix     string
}

// NewLogger creates a new logger instance
func NewLogger(prefix string, level Level) *Logger {

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	return &Logger{
		level:  level,
		prefix: prefix,
		logger: log.New(os.Stdout, fmt.Sprintf("[%s] ", prefix), log.LstdFlags|log.Lshortfile),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the output destination
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.SetOutput(w)
}

func (l *Logger) log(level Level, v ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf("[%s] %s", level.String(), fmt.Sprint(v...))
	l.logger.Output(3, msg)
}

func (l *Logger) logf(level Level, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf("[%s] %s", level.String(), fmt.Sprintf(format, v...))
	l.logger.Output(3, msg)
}

func (l *Logger) Debug(v ...interface{}) { l.log(DEBUG, v...) }
func (l *Logger) Info(v ...interface{})  { l.log(INFO, v...) }
func (l *Logger) Warn(v ...interface{})  { l.log(WARN, v...) }
func (l *Logger) Error(v ...interface{}) { l.log(ERROR, v...) }
func (l *Logger) Fatal(v ...interface{}) {
	l.log(FATAL, v...)
	os.Exit(1)
}

func (l *Logger) Debugf(format string, v ...interface{}) { l.logf(DEBUG, format, v...) }
func (l *Logger) Infof(format string, v ...interface{})  { l.logf(INFO, format, v...) }
func (l *Logger) Warnf(format string, v ...interface{})  { l.logf(WARN, format, v...) }
func (l *Logger) Errorf(format string, v ...interface{}) { l.logf(ERROR, format, v...) }
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logf(FATAL, format, v...)
	os.Exit(1)
}

// Global logger instance
var defaultLogger = NewLogger("APP", INFO)

// Package-level convenience functions
func SetLevel(level Level)                   { defaultLogger.SetLevel(level) }
func SetOutput(w io.Writer)                  { defaultLogger.SetOutput(w) }
func Debug(v ...interface{})                 { defaultLogger.Debug(v...) }
func Info(v ...interface{})                  { defaultLogger.Info(v...) }
func Warn(v ...interface{})                  { defaultLogger.Warn(v...) }
func Error(v ...interface{})                 { defaultLogger.Error(v...) }
func Fatal(v ...interface{})                 { defaultLogger.Fatal(v...) }
func Debugf(format string, v ...interface{}) { defaultLogger.Debugf(format, v...) }
func Infof(format string, v ...interface{})  { defaultLogger.Infof(format, v...) }
func Warnf(format string, v ...interface{})  { defaultLogger.Warnf(format, v...) }
func Errorf(format string, v ...interface{}) { defaultLogger.Errorf(format, v...) }
func Fatalf(format string, v ...interface{}) { defaultLogger.Fatalf(format, v...) }
