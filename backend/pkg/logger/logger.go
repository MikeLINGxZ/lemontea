package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	rdebug "runtime/debug"
	"sync"
	"time"

	"github.com/fatih/color"
)

const debugFormat = "DEBUG [%s]: \n%s"
const formatWithName = "%s | %s | [%s]: %s"
const format = "%s | %s | %s"

type Logger struct {
	loggerName  string
	errorLogger *color.Color
	warmLogger  *color.Color
	infoLogger  *color.Color
	debugLogger *color.Color
	panicLogger *color.Color
	logger      func(color *color.Color, level, debug, content string, any ...any)
	esIndex     *string
	enableDebug bool
	logDir      string
	currentDate string
	logFile     *os.File
	fileMutex   sync.Mutex
}

func NewLogger(name string, options ...Options) (*Logger, error) {
	errorLogger := color.New(color.FgRed)
	warmLogger := color.New(color.FgYellow)
	normalLogger := color.New(color.FgCyan)
	debugLogger := color.New(color.FgMagenta)
	panicLogger := color.New(color.FgRed)
	logger := &Logger{
		loggerName:  name,
		errorLogger: errorLogger,
		warmLogger:  warmLogger,
		debugLogger: debugLogger,
		infoLogger:  normalLogger,
		panicLogger: panicLogger,
	}
	logger.logger = logger.standerLog
	for _, option := range options {
		err := option(logger)
		if err != nil {
			return nil, err
		}
	}
	return logger, nil
}

func (l *Logger) Infof(content string, any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.infoLogger, "INFO", debugInfo, content, any...)
}

func (l *Logger) Warmf(content string, any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.warmLogger, "WARM", debugInfo, content, any...)
}

func (l *Logger) Errorf(content string, any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.errorLogger, "ERROR", debugInfo, content, any...)
}

func (l *Logger) Debugf(content string, any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.debugLogger, "DEBUG", debugInfo, content, any...)
}

func (l *Logger) Panicf(content string, any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.panicLogger, "PANIC", debugInfo, content, any...)
	panic(content)
}

func (l *Logger) Info(any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.infoLogger, "INFO", debugInfo, "", any...)
}

func (l *Logger) Warm(any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.warmLogger, "WARM", debugInfo, "", any...)
}

func (l *Logger) Error(any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.errorLogger, "ERROR", debugInfo, "", any...)
}

func (l *Logger) Debug(any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.debugLogger, "DEBUG", debugInfo, "", any...)
}

func (l *Logger) Panic(any ...any) {
	debugInfo := l.callerInfo(runtime.Caller(1))
	l.logger(l.panicLogger, "PANIC", debugInfo, "", any...)
	os.Exit(-1)
}

func (l *Logger) callerInfo(pc uintptr, file string, line int, ok bool) string {
	if !ok {
		return ""
	}
	callerFunction := runtime.FuncForPC(pc)
	if callerFunction != nil {
		return fmt.Sprintf("file_uploader: %s\nline: %d\nfunction: %s\n", file, line, callerFunction.Name())
	}
	return fmt.Sprintf("file_uploader: %s\nline: %d\n", file, line)
}

func (l *Logger) standerLog(color *color.Color, level, debug, content string, any ...any) {
	logStr := ""
	if content == "" {
		logStr = fmt.Sprint(any...)
	} else {
		logStr = fmt.Sprintf(content, any...)
	}
	if l.loggerName != "" {
		_, err := color.Println(fmt.Sprintf(formatWithName, time.Now().Format(time.DateTime), level, l.loggerName, logStr))
		if err != nil {
			fmt.Println(err)
		}
	} else {
		_, err := color.Println(fmt.Sprintf(format, time.Now().Format(time.DateTime), level, logStr))
		if err != nil {
			fmt.Println(err)
		}
	}

	l.writeToFile(level, logStr)

	if level == "ERROR" || level == "PANIC" {
		stack := rdebug.Stack()
		_, err := color.Println(string(stack))
		if err != nil {
			fmt.Println(err)
		}
		l.writeToFile(level, string(stack))
	}

	if !l.enableDebug {
		return
	}
	if debug == "" {
		debug = "-"
	}
	_, err := l.debugLogger.Println(fmt.Sprintf(debugFormat, l.loggerName, debug))
	if err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) writeToFile(level, logStr string) {
	if l.logDir == "" {
		return
	}

	l.fileMutex.Lock()
	defer l.fileMutex.Unlock()

	today := time.Now().Format(time.DateOnly)
	if l.currentDate != today {
		if l.logFile != nil {
			l.logFile.Close()
			l.logFile = nil
		}

		logFilePath := filepath.Join(l.logDir, today+".log")
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file: %v\n", err)
			return
		}
		l.logFile = file
		l.currentDate = today
	}

	if l.logFile == nil {
		return
	}

	var logLine string
	if l.loggerName != "" {
		logLine = fmt.Sprintf(formatWithName, time.Now().Format(time.DateTime), level, l.loggerName, logStr)
	} else {
		logLine = fmt.Sprintf(format, time.Now().Format(time.DateTime), level, logStr)
	}

	_, err := l.logFile.WriteString(logLine + "\n")
	if err != nil {
		fmt.Printf("Failed to write to log file: %v\n", err)
	}
}

type LogFormat struct {
	Name     string `json:"name"`
	LogLevel string `json:"log_level"`
	Content  string `json:"content"`
	Time     string `json:"time"`
	Debug    string `json:"debug"`
}
