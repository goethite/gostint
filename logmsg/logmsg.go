package logmsg

import (
	"fmt"
	"log"

	"github.com/fatih/color"
)

var enableDebug bool

// EnableDebug enable debugging
func EnableDebug() {
	enableDebug = true
}

// DisableDebug disable debugging
func DisableDebug() {
	enableDebug = false
}

// Error log an error
func Error(format string, a ...interface{}) {
	format = "ERROR: " + format
	colmsg := fmt.Sprintf(format, a...)
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	log.Printf(red(colmsg))
}

// Warn log a warning
func Warn(format string, a ...interface{}) {
	format = "WARN: " + format
	colmsg := fmt.Sprintf(format, a...)
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	log.Printf(yellow(colmsg))
}

// Info log an Info msg
func Info(format string, a ...interface{}) {
	format = "INFO: " + format
	colmsg := fmt.Sprintf(format, a...)
	green := color.New(color.FgGreen).SprintFunc()
	log.Printf(green(colmsg))
}

// Debug log a debug msg
func Debug(format string, a ...interface{}) {
	if !enableDebug {
		return
	}
	debug(format, a...)
}

func debug(format string, a ...interface{}) {
	format = "DEBUG: " + format
	colmsg := fmt.Sprintf(format, a...)
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	log.Printf(cyan(colmsg))
}
