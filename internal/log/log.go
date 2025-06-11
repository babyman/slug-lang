package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Level int

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	NONE
)

var levelNames = [...]string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "NONE"}

var levelColors = [...]string{
	"\033[90m", // Grey
	"\033[36m", // Cyan
	"\033[32m", // Green
	"\033[33m", // Yellow
	"\033[31m", // Red
}

const resetColor = "\033[0m"

type Logger struct {
	level      Level
	color      bool
	toFile     bool
	fileHandle *os.File
	logger     *log.Logger
	mu         sync.Mutex
}

var Log *Logger

func InitLogger(
	logLevel string,
	logFile string,
	color bool,
) {
	lvl := parseLevel(logLevel)
	var out io.Writer = os.Stderr
	var fh *os.File

	if logFile != "" {
		var err error
		fh, err = os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		} else {
			out = fh
		}
	}

	Log = &Logger{
		level:      lvl,
		color:      color && isTerminal(out),
		toFile:     fh != nil,
		fileHandle: fh,
		logger:     log.New(out, "", log.LstdFlags),
	}

	Log.setupLogRotation(logFile)
}

func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, _ := f.Stat()
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

func parseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "trace":
		return TRACE
	case "debug":
		return DEBUG
	case "warn":
		return WARN
	case "error":
		return ERROR
	case "info":
		return INFO
	default:
		return NONE
	}
}

func (l *Logger) log(level Level, format string, v ...any) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, v...)
	tag := levelNames[level]
	if l.color {
		tag = fmt.Sprintf("%s%-5s%s", levelColors[level], tag, resetColor)
	}
	l.logger.Printf("[%s] %s", tag, msg)
}

func (l *Logger) reopenLogFile(path string) {
	if l.fileHandle != nil {
		l.fileHandle.Close()
	}
	var err error
	l.fileHandle, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.logger.Fatalf("could not reopen log file: %v", err)
	}
	l.logger.SetOutput(l.fileHandle)
}

func (l *Logger) setupLogRotation(path string) {
	if !l.toFile {
		return
	}

	/*
	 * if we're logging to a file listen for SIGHUP on log file rotation
	 * ps aux | grep slug
	 * mv slug.log slug.bak && kill -HUP 59088
	 */
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	go func() {
		for range sigs {
			l.reopenLogFile(path)
		}
	}()
}

func Trace(format string, v ...any) { Log.log(TRACE, format, v...) }
func Debug(format string, v ...any) { Log.log(DEBUG, format, v...) }
func Info(format string, v ...any)  { Log.log(INFO, format, v...) }
func Warn(format string, v ...any)  { Log.log(WARN, format, v...) }
func Error(format string, v ...any) { Log.log(ERROR, format, v...) }

func Close() {
	if Log != nil && Log.fileHandle != nil {
		_ = Log.fileHandle.Close()
	}
}
