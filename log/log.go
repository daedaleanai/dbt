package log

import (
	"fmt"
	"os"
	"strings"
)

// Verbose controls whether debug messages are being printed.
var Verbose bool

// NoColor controls whether stdout and stderr are colorized or not
var NoColor bool

// IndentationLevel controls the amount of indentation of log messages.
var IndentationLevel = 0

var errorOccured = false

type Color uint

const (
	ColorReset Color = iota
	ColorBlue
	ColorRed
	ColorGreen
	ColorYellow
)

func GetColorString(color Color) string {
	if NoColor {
		return ""
	}

	switch color {
	case ColorReset:
		return "\033[0m"
	case ColorBlue:
		return "\033[36m"
	case ColorRed:
		return "\033[31m"
	case ColorGreen:
		return "\033[32m"
	case ColorYellow:
		return "\033[33m"
	}

	return ""
}

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
		fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+GetColorString(ColorBlue)+"Debug: "+GetColorString(ColorReset)+format, a...)
	}
}

// Success prints an indented and formatted success message to os.Stdout.
func Success(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+GetColorString(ColorGreen)+"Success: "+GetColorString(ColorReset)+format, a...)
}

// Warning prints an indented and formatted warning to os.Stdout.
func Warning(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+GetColorString(ColorYellow)+"Warning: "+GetColorString(ColorReset)+format, a...)
}

// Error prints an indented and formatted error message to os.Stdout.
func Error(format string, a ...interface{}) {
	errorOccured = true
	fmt.Fprintf(os.Stderr, strings.Repeat("  ", IndentationLevel)+GetColorString(ColorRed)+"Error: "+GetColorString(ColorReset)+format, a...)
}

// Fatal prints an indented and formatted error message to os.Stdout and terminates the program.
func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	fmt.Fprintf(os.Stderr, GetColorString(ColorRed)+"A fatal error occured. Exiting..."+GetColorString(ColorReset)+"\n")
	os.Exit(1)
}
