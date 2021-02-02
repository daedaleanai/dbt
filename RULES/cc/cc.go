package cc

import (
	"fmt"
	"strings"

	"dbt/RULES/core"
)

// Toolchain represents a C++ toolchain.
type Toolchain struct {
	Cc     core.GlobalFile
	Ar     core.GlobalFile
	CFlags []string
}

var defaultToolchain = Toolchain{
	Cc: core.NewGlobalFile("gcc"),
	Ar: core.NewGlobalFile("ar"),
}

// ObjectFile compiles a single C++ source file.
type ObjectFile struct {
	Src       core.File
	Includes  core.Files
	Toolchain *Toolchain
}

// Out provides the name of the output created by ObjectFile.
func (obj ObjectFile) Out() core.OutFile {
	return obj.Src.WithExt("o")
}

// BuildSteps provides the steps to produce an ObjectFile.
func (obj ObjectFile) BuildSteps() []core.BuildStep {
	if obj.Toolchain == nil {
		obj.Toolchain = &defaultToolchain
	}

	includes := strings.Builder{}
	for _, include := range obj.Includes {
		includes.WriteString(fmt.Sprintf("-I%s ", include))
	}
	flags := strings.Join(obj.Toolchain.CFlags, " ")
	cmd := fmt.Sprintf("%s -c %s %s -o %s %s", obj.Toolchain.Cc, flags, includes.String(), obj.Out(), obj.Src)
	return []core.BuildStep{{
		Out:   obj.Out(),
		In:    obj.Src,
		Cmd:   cmd,
		Descr: fmt.Sprintf("CC %s", obj.Out().RelPath()),
		Alias: obj.Out().RelPath(),
	}}
}

// Library builds and links a C++ library.
type Library struct {
	Out       core.OutFile
	Srcs      core.Files
	CFlags    []string
	Includes  core.Files
	Deps      []Library
	Toolchain *Toolchain
}

// BuildSteps provides the steps to build a Library.
func (lib Library) BuildSteps() []core.BuildStep {
	core.Assert(!lib.Out.Empty(), "'Out' is missing, but required")

	if lib.Toolchain == nil {
		lib.Toolchain = &defaultToolchain
	}

	var steps = []core.BuildStep{}
	var objs = core.Files{}

	for _, src := range lib.Srcs {
		obj := ObjectFile{
			Src:       src,
			Includes:  lib.Includes,
			Toolchain: lib.Toolchain,
		}
		objs = append(objs, obj.Out())
		steps = append(steps, obj.BuildSteps()...)
	}

	cmd := fmt.Sprintf("%s rv %s %s > /dev/null 2> /dev/null", lib.Toolchain.Ar, lib.Out, objs)
	linkStep := core.BuildStep{
		Out:   lib.Out,
		Ins:   objs,
		Cmd:   cmd,
		Descr: fmt.Sprintf("AR %s", lib.Out.RelPath()),
		Alias: lib.Out.RelPath(),
	}

	return append(steps, linkStep)
}
