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

// SourceDir returns the workspace source directory.
func SourceDir() string {
	return os.Args[2]
}

// BuildDir returns the workspace build directory.
func BuildDir() string {
	return os.Args[3]
}

// WorkingDir returns the directory the build command was executed in.
func WorkingDir() string {
	return os.Args[4]
}

var reportedFlags = map[string]struct{}{}

// Flag provides the value of a build config flags.
func Flag(name string) string {
	if mode() == "flags" {
		_, exists := reportedFlags[name]
		if !exists {
			reportedFlags[name] = struct{}{}
			fmt.Printf("--%s\n", name)
		}
	}

	prefix := fmt.Sprintf("--%s=", name)
	for _, arg := range os.Args[4:] {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
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
	fmt.Fprintf(os.Stderr, "A fatal error occured while processing target '%s': %s", currentTarget, msg)
	os.Exit(1)
}

// Assert can be used in build rules to abort build file generation with an error message if `cond` is true.
func Assert(cond bool, format string, a ...interface{}) {
	// Ignore all asserts when not generating the ninja build file. This allows listing all targets in a workspace
	// without specifying required build flags.
	if mode() != "ninja" {
		return
	}
	if !cond {
		msg := fmt.Sprintf(format, a...)
		fmt.Fprintf(os.Stderr, "Assertion failed while processing target '%s': %s", currentTarget, msg)
		os.Exit(1)
	}
}
