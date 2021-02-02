package core

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// CurrentModulePath holds the path of the current module relative to the workspace directory.
var CurrentModulePath string

// CurrentTargetPath holds the path of the current target relative to the workspace directory.
var CurrentTargetPath string

// CurrentTargetName holds the name of the current target.
var CurrentTargetName string

const depsDirName = "DEPS"

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

// Assert can be used in build rules to abort buildfile generation with an error message.
func Assert(cond bool, msg string) {
	if !cond {
		currentTarget := path.Join(CurrentTargetPath, CurrentTargetName)
		fmt.Fprintf(os.Stderr, "Assertion failed while processing target '%s': %s.\n", currentTarget, msg)
		os.Exit(1)
	}
}

func GetWorkspaceSourceDir() string {
	return os.Args[1]
}

func GetWorkspaceBuildDir() string {
	return os.Args[2]
}

func GetDepsSourceDir() string {
	return path.Join(GetWorkspaceSourceDir(), depsDirName)
}

func GetDepsBuildDir() string {
	return path.Join(GetWorkspaceBuildDir(), depsDirName)
}

func GetModuleSourceDir() string {
	return path.Join(GetWorkspaceSourceDir(), CurrentModulePath)
}

func GetModuleBuildDir() string {
	return path.Join(GetWorkspaceBuildDir(), CurrentModulePath)
}

func GetTargetSourceDir() string {
	return path.Join(GetWorkspaceSourceDir(), CurrentTargetPath)
}

func GetTargetBuildDir() string {
	return path.Join(GetWorkspaceBuildDir(), CurrentTargetPath)
}
