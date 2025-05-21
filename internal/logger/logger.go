package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
)

var (
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
	debugLog *log.Logger

	debugEnabled = false // set ture for enable debug logging
)

// initializes the loggers. Automatically called when the package is imported
func init() {
	// Common flags for all loggers
	// Ldate: date YYYY/MM/DD
	// Ltime: time HH:MM:SS
	// Lmicroseconds: include microseconds
	baseFlags := log.Ldate | log.Ltime | log.Lmicroseconds

	infoLog = log.New(os.Stdout, "INFO: ", baseFlags)
	warnLog = log.New(os.Stdout, "WARN: ", baseFlags) // os.Stdout for warnings
	errorLog = log.New(os.Stderr, "ERROR: ", baseFlags)
	debugLog = log.New(os.Stdout, "DEBUG: ", baseFlags)
}

// return file and line number of the caller
func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	// get just file name
	parts := strings.Split(file, "/")
	fileName := parts[len(parts)-1]

	return fmt.Sprintf("%s:%d", fileName, line)
}

// Info Logs
func Info(format string, v ...interface{}) {
	caller := getCallerInfo(2)
	message := fmt.Sprintf(format, v...)
	infoLog.Printf("%s: %s", caller, message)
}

// Warning Logs
func Warn(format string, v ...interface{}) {
	caller := getCallerInfo(2)
	message := fmt.Sprintf(format, v...)
	warnLog.Printf("%s: %s", caller, message)
}

// Error logs
func Error(format string, v ...interface{}) {
	caller := getCallerInfo(2)
	message := fmt.Sprintf(format, v...)
	errorLog.Printf("%s: %s", caller, message)
}

// If debug enabled
func Debug(format string, v ...interface{}) {
	if debugEnabled {
		caller := getCallerInfo(2)
		message := fmt.Sprintf(format, v...)
		debugLog.Printf("%s: %s", caller, message)
	}
}

// Fatal Logs calls os.Exit(1)
func Fatal(format string, v ...interface{}) {
	caller := getCallerInfo(2)
	message := fmt.Sprintf(format, v...)
	errorLog.Printf("%s: %s", caller, message)
	os.Exit(1)
}

func SetDebug(enable bool) {
	debugEnabled = enable
}
