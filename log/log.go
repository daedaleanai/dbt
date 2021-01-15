package log

import (
	"fmt"
	"os"
	"strings"
)

// Log prints an indented and formatted message to os.Stdout.
func Log(level int, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("   ", level)+format, a...)
}

// Success prints an indented and formatted success message to os.Stdout.
func Success(level int, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("   ", level)+"\033[32mSuccess: \033[0m"+format, a...)
}

// Warn prints an indented and formatted warning to os.Stdout.
func Warn(level int, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("   ", level)+"\033[33mWarning: \033[0m"+format, a...)
}

// Error prints an indented and formatted error message to os.Stdout and terminates the program.
func Error(level int, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.Repeat("   ", level)+"\033[31mError: \033[0m"+format, a...)
	fmt.Fprintf(os.Stderr, "\033[31mAn error occured. Exiting...\033[0m\n")
	os.Exit(1)
}
