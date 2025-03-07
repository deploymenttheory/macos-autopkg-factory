package logger

import (
	"fmt"
	"sync"
)

// Log levels
const (
	LogDebug   = 10
	LogInfo    = 20
	LogWarning = 30
	LogError   = 40
	LogSuccess = 50
)

// Global log level setting with thread-safe access
var (
	currentLogLevel = LogInfo
	logMutex        sync.RWMutex
)

// SetLogLevel sets the minimum log level that will be displayed
func SetLogLevel(level int) {
	logMutex.Lock()
	defer logMutex.Unlock()
	currentLogLevel = level
}

// GetLogLevel returns the current log level
func GetLogLevel() int {
	logMutex.RLock()
	defer logMutex.RUnlock()
	return currentLogLevel
}

// Logger implements a simple logging system that respects the current log level
func Logger(message string, level int) {
	logMutex.RLock()
	shouldLog := level >= currentLogLevel
	logMutex.RUnlock()

	if !shouldLog {
		return
	}

	var prefix string
	switch level {
	case LogDebug:
		prefix = "[DEBUG] "
	case LogInfo:
		prefix = "[INFO] "
	case LogWarning:
		prefix = "[WARNING] "
	case LogError:
		prefix = "[ERROR] "
	case LogSuccess:
		prefix = "[SUCCESS] "
	default:
		prefix = "[LOG] "
	}
	fmt.Println(prefix + message)
}

// Debug logs a debug message
func Debug(message string) {
	Logger(message, LogDebug)
}

// Info logs an info message
func Info(message string) {
	Logger(message, LogInfo)
}

// Warning logs a warning message
func Warning(message string) {
	Logger(message, LogWarning)
}

// Error logs an error message
func Error(message string) {
	Logger(message, LogError)
}

// Success logs a success message
func Success(message string) {
	Logger(message, LogSuccess)
}
