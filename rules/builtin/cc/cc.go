package cc

import (
	"fmt"
	"strings"

	"_/builtin/core"
)

// Toolchain represents a C++ toolchain.
type Toolchain struct {
	Cc string
	Ar string
}

var defaultToolchain = Toolchain{
	Cc: "gcc",
	Ar: "ar",
}

// Flags or compiling and linking C++ targets.
type Flags struct {
	Includes       []string
	SystemIncludes []string
	CFlags         []string
	LdFlags        []string
}

func (flags *Flags) compileFlags() string {
	cflags := []string{"-c"}
	cflags = append(cflags, flags.CFlags...)
	for _, include := range flags.Includes {
		cflags = append(cflags, fmt.Sprintf("-I%s", include))
	}
	for _, include := range flags.SystemIncludes {
		cflags = append(cflags, fmt.Sprintf("-isystem %s", include))
	}
	return strings.Join(cflags, " ")
}

// ObjectFile compiles a single C++ source file.
type ObjectFile struct {
	Src       core.File
	Flags     Flags
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

	cmd := fmt.Sprintf("%s %s -o %s %s", obj.Toolchain.Cc, obj.Flags.CompileFlags(), obj.Out(), obj.Src)
	return core.BuildSteps{{
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
	Deps      []Library
	Toolchain *Toolchain
}

// BuildSteps provides the steps to build a Library.
func (lib Library) BuildSteps() core.BuildSteps {
	core.Assert(!lib.Out.Empty(), "'Out' is missing, but required", target)

	if lib.Toolchain == nil {
		lib.Toolchain = &defaultToolchain
	}

	var steps = core.BuildSteps{}
	var objs = core.Files{}

	for _, src := range lib.Srcs {
		obj := ObjectFile{
			Src:       src,
			Toolchain: lib.Toolchain,
		}
		objs = append(objs, obj.Out())
		steps = append(steps, obj.BuildSteps()...)
	}

	cmd := fmt.Sprintf("%s rv %s %s > /dev/null", lib.Toolchain.Ar, lib.Out, objs)
	linkStep := core.BuildStep{
		Out:   lib.Out,
		Ins:   objs,
		Cmd:   cmd,
		Descr: fmt.Sprintf("AR %s", lib.Out.RelPath()),
		Alias: lib.Out.RelPath(),
	}

	return append(steps, linkStep)
}
