package cc

import (
	"fmt"
	"strings"

	"dbt/RULES/core"
)

// Toolchain represents a C++ toolchain.
type Toolchain struct {
	Ar      core.GlobalFile
	As      core.GlobalFile
	Cc      core.GlobalFile
	Cpp     core.GlobalFile
	Cxx     core.GlobalFile
	Objcopy core.GlobalFile

	Includes core.Files
}

var defaultToolchain = Toolchain{
	Ar:      core.NewGlobalFile("ar"),
	As:      core.NewGlobalFile("as"),
	Cc:      core.NewGlobalFile("gcc"),
	Cpp:     core.NewGlobalFile("g++"),
	Cxx:     core.NewGlobalFile("gcc"),
	Objcopy: core.NewGlobalFile("objcopy"),
}

// ObjectFile compiles a single C++ source file.
type ObjectFile struct {
	Src       core.File
	Includes  core.Files
	CFlags    []string
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
	for _, include := range obj.Toolchain.Includes {
		includes.WriteString(fmt.Sprintf("-isystem %s ", include))
	}
	depfile := obj.Src.WithExt("d")
	flags := strings.Join(obj.CFlags, " ")
	cmd := fmt.Sprintf("%s -c -MD -MF %s %s %s -o %s %s", obj.Toolchain.Cc, depfile, flags, includes.String(), obj.Out(), obj.Src)
	return []core.BuildStep{{
		Out:     obj.Out(),
		Depfile: &depfile,
		In:      obj.Src,
		Cmd:     cmd,
		Descr:   fmt.Sprintf("CC %s", obj.Out().RelPath()),
		Alias:   obj.Out().RelPath(),
	}}
}

// Library builds and links a C++ library.
type Library struct {
	Out       core.OutFile
	Srcs      core.Files
	Includes  core.Files
	CFlags    []string
	Toolchain *Toolchain
}

// BuildSteps provides the steps to build a Library.
func (lib Library) BuildSteps() []core.BuildStep {
	if lib.Toolchain == nil {
		lib.Toolchain = &defaultToolchain
	}

	lib.Includes = append(lib.Includes, core.NewInFile("."))

	var steps = []core.BuildStep{}
	var objs = core.Files{}

	for _, src := range lib.Srcs {
		obj := ObjectFile{
			Src:       src,
			Includes:  lib.Includes,
			CFlags:    lib.CFlags,
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
