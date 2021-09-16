package log

import (
	"fmt"
	"os"
	"strings"
)

// Verbose controls whether debug messages are being printed.
var Verbose bool

// IndentationLevel controls the amount of indentation of log messages.
var IndentationLevel = 0

var errorOccured = false

// ErrorOccured reports whether any errors have occured.
func ErrorOccured() bool {
	return errorOccured
}

// Log prints an indented and formatted message to os.Stdout.
func Log(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+format, a...)
}

// Debug prints an indented and formatted debug message to os.Stdout if verbose output is selected.
func Debug(format string, a ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+"\033[36mDebug: \033[0m"+format, a...)
	}
}

// Success prints an indented and formatted success message to os.Stdout.
func Success(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+"\033[32mSuccess: \033[0m"+format, a...)
}

// Warning prints an indented and formatted warning to os.Stdout.
func Warning(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+"\033[33mWarning: \033[0m"+format, a...)
}

// Error prints an indented and formatted error message to os.Stdout.
func Error(format string, a ...interface{}) {
	errorOccured = true
	fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+"\033[31mError: \033[0m"+format, a...)
}

// Fatal prints an indented and formatted error message to os.Stdout and terminates the program.
func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	fmt.Fprintf(os.Stderr, "\033[31mA fatal error occured. Exiting...\033[0m\n")
	os.Exit(1)
}
