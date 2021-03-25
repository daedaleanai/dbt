package core

import (
	"fmt"
	"os"
	"strings"
)

var currentTarget string

func mode() string {
	return os.Args[1]
}

func sourceDir() string {
	return os.Args[2]
}

func buildDir() string {
	return os.Args[3]
}

func workingDir() string {
	return os.Args[4]
}

var allowNewFlags = true
var BuildFlags = map[string]string{}

func flagValue(name string) string {
	prefix := fmt.Sprintf("%s=", name)
	for _, arg := range os.Args[4:] {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

// LockBuildFlags prevents new flags from being used.
func LockBuildFlags() {
	allowNewFlags = false
}

// Flag provides the value of a build config flags.
func Flag(name string) string {
	if allowNewFlags {
		BuildFlags[name] = flagValue(name)
	}

	if value, exists := BuildFlags[name]; exists {
		return value
	}

	Fatal("Tried to use flag '%s' after flags were locked. Flags must be accessed outside of build rule definitions.", name)
	return ""
}

// Fatal can be used in build rules to abort build file generation with an error message unconditionally.
func Fatal(format string, a ...interface{}) {
	// Ignore all errors when not generating the ninja build file. This allows listing all targets in a workspace
	// without specifying required build flags.
	if mode() != "ninja" {
		return
	}
	msg := fmt.Sprintf(format, a...)
	if currentTarget == "" {
		fmt.Fprintf(os.Stderr, "A fatal error occured: %s", msg)
	} else {
		fmt.Fprintf(os.Stderr, "A fatal error occured while processing target '%s': %s", currentTarget, msg)
	}
	os.Exit(1)
}

// Assert can be used in build rules to abort build file generation with an error message if `cond` is true.
func Assert(cond bool, format string, a ...interface{}) {
	// Ignore all asserts when not generating the ninja build file. This allows listing all targets in a workspace
	// without specifying required build flags.
	if cond || mode() != "ninja" {
		return
	}
	msg := fmt.Sprintf(format, a...)
	if currentTarget == "" {
		fmt.Fprintf(os.Stderr, "Assertion failed: %s", msg)
	} else {
		fmt.Fprintf(os.Stderr, "Assertion failed while processing target '%s': %s", currentTarget, msg)
	}
	os.Exit(1)
}
