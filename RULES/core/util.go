package core

import (
	"fmt"
	"os"
	"strings"
)

// CurrentTarget holds the current target relative to the workspace directory.
var CurrentTarget string

func SourceDir() string {
	return os.Args[1]
}

func BuildDir() string {
	return os.Args[2]
}

// Flag provides values of build flags.
func Flag(name string) string {
	prefix := fmt.Sprintf("--%s=", name)
	for _, arg := range os.Args[3:] {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

// Fatal can be used in build rules to abort buildfile generation with an error message unconditionally.
func Fatal(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "A fatal error occured while processing target '%s': %s", CurrentTarget, msg)
	os.Exit(1)
}

// Assert can be used in build rules to abort buildfile generation with an error message.
func Assert(cond bool, format string, a ...interface{}) {
	if !cond {
		msg := fmt.Sprintf(format, a...)
		fmt.Fprintf(os.Stderr, "Assertion failed while processing target '%s': %s", CurrentTarget, msg)
		os.Exit(1)
	}
}
