package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type Level int

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

func ParseLevel(level string) Level {
	switch strings.ToUpper(level) {
	case "TRACE":
		return TRACE
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return FATAL
	}
}

func (l Level) String() string {
	switch l {
	case TRACE:
		return "TRACE"
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

// ColoredString returns the level string with appropriate color codes
func (l Level) ColoredString() string {
	switch l {
	case TRACE:
		return colorGray + "TRACE" + colorReset
	case DEBUG:
		return colorCyan + "DEBUG" + colorReset
	case INFO:
		return colorGreen + "INFO" + colorReset
	case WARN:
		return colorYellow + "WARN" + colorReset
	case ERROR:
		return colorRed + "ERROR" + colorReset
	case FATAL:
		return colorBold + colorRed + "FATAL" + colorReset
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	level      Level
	toFile     bool
	fileHandle *os.File
	logger     *log.Logger
	mu         sync.Mutex
	prefix     string
}

// NewLogger creates a new logger instance
func NewLogger(prefix string, level Level) *Logger {
	return &Logger{
		level:  level,
		prefix: prefix,
		logger: log.New(os.Stdout, fmt.Sprintf("[%s] ", prefix), log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// isOutputToTerminal checks if the current output is a terminal (stdout)
func (l *Logger) isOutputToTerminal() bool {
	return !l.toFile
}

// SetLogFile switches logging to a file
func (l *Logger) SetLogFile(filepath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close existing file handle if any
	if l.fileHandle != nil {
		l.fileHandle.Close()
	}

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.fileHandle = file
	l.toFile = true
	l.logger.SetOutput(file)
	return nil
}

// SetConsoleOutput switches logging back to console
func (l *Logger) SetConsoleOutput() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close file handle if any
	if l.fileHandle != nil {
		l.fileHandle.Close()
		l.fileHandle = nil
	}

	l.toFile = false
	l.logger.SetOutput(os.Stdout)
}

// Close properly closes the logger and any open file handles
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileHandle != nil {
		err := l.fileHandle.Close()
		l.fileHandle = nil
		return err
	}
	return nil
}

func (l *Logger) log(level Level, v ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var levelStr string
	if l.isOutputToTerminal() {
		levelStr = level.ColoredString()
	} else {
		levelStr = level.String()
	}

	msg := fmt.Sprintf("[%s] %s", levelStr, fmt.Sprint(v...))
	l.logger.Output(3, msg)
}

func (l *Logger) logf(level Level, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var levelStr string
	if l.isOutputToTerminal() {
		levelStr = level.ColoredString()
	} else {
		levelStr = level.String()
	}

	msg := fmt.Sprintf("[%s] %s", levelStr, fmt.Sprintf(format, v...))
	l.logger.Output(3, msg)
}

func (l *Logger) Trace(v ...interface{}) { l.log(TRACE, v...) }
func (l *Logger) Debug(v ...interface{}) { l.log(DEBUG, v...) }
func (l *Logger) Info(v ...interface{})  { l.log(INFO, v...) }
func (l *Logger) Warn(v ...interface{})  { l.log(WARN, v...) }
func (l *Logger) Error(v ...interface{}) { l.log(ERROR, v...) }
func (l *Logger) Fatal(v ...interface{}) {
	l.log(FATAL, v...)
	os.Exit(1)
}

func (l *Logger) Tracef(format string, v ...interface{}) { l.logf(TRACE, format, v...) }
func (l *Logger) Debugf(format string, v ...interface{}) { l.logf(DEBUG, format, v...) }
func (l *Logger) Infof(format string, v ...interface{})  { l.logf(INFO, format, v...) }
func (l *Logger) Warnf(format string, v ...interface{})  { l.logf(WARN, format, v...) }
func (l *Logger) Errorf(format string, v ...interface{}) { l.logf(ERROR, format, v...) }
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logf(FATAL, format, v...)
	os.Exit(1)
}
