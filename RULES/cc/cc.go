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
	Src            core.File
	Includes       core.Files
	SystemIncludes core.Files
	Flags          []string
	Toolchain      *Toolchain
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
	cmd := fmt.Sprintf("%s -c -MD -MF %s %s %s -o %s %s", obj.Toolchain.Cxx, depfile, strings.Join(obj.Flags, " "), includes.String(), obj.Out(), obj.Src)
	return []core.BuildStep{{
		Out:     obj.Out(),
		Depfile: &depfile,
		In:      obj.Src,
		Cmd:     cmd,
		Descr:   fmt.Sprintf("CC %s", obj.Out().RelPath()),
		Alias:   obj.Out().RelPath(),
	}}
}

func flattenDeps(deps []Library) []Library {
	allDeps := append([]Library{}, deps...)
	for _, dep := range deps {
		allDeps = append(allDeps, flattenDeps(dep.Deps)...)
	}
	return allDeps
}

func compileSources(srcs core.Files, flags []string, deps []Library, toolchain *Toolchain) ([]core.BuildStep, core.Files) {
	deps = flattenDeps(deps)

	includes := core.Files{core.NewInFile(".")}
	for _, dep := range deps {
		includes = append(includes, dep.Includes...)
	}

	steps := []core.BuildStep{}
	objs := core.Files{}

	for _, src := range srcs {
		obj := ObjectFile{
			Src:       src,
			Includes:  includes,
			Flags:     flags,
			Toolchain: toolchain,
		}
		objs = append(objs, obj.Out())
		steps = append(steps, obj.BuildSteps()...)
	}

	return steps, objs
}

// Library builds and links a C++ library.
type Library struct {
	Out       core.OutFile
	Srcs      core.Files
	Includes  core.Files
	CxxFlags  []string
	Deps      []Library
	Toolchain *Toolchain
}

// BuildSteps provides the steps to build a Library.
func (lib Library) BuildSteps() []core.BuildStep {
	toolchain := lib.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	steps, objs := compileSources(lib.Srcs, lib.CxxFlags, []Library{lib}, toolchain)

	cmd := fmt.Sprintf("%s rv %s %s > /dev/null 2> /dev/null", toolchain.Ar, lib.Out, objs)
	arStep := core.BuildStep{
		Out:   lib.Out,
		Ins:   objs,
		Cmd:   cmd,
		Descr: fmt.Sprintf("AR %s", lib.Out.RelPath()),
		Alias: lib.Out.RelPath(),
	}

	return append(steps, arStep)
}

type Binary struct {
	Out       core.OutFile
	Srcs      core.Files
	CxxFlags  []string
	LdFlags   []string
	Deps      []Library
	Script    *core.File
	Toolchain *Toolchain
}

// BuildSteps provides the steps to build a Binary.
func (bin Binary) BuildSteps() []core.BuildStep {
	toolchain := bin.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	steps, objs := compileSources(bin.Srcs, bin.CxxFlags, bin.Deps, toolchain)

	flags := bin.LdFlags
	if bin.Script != nil {
		flags = append(flags, fmt.Sprintf("-T%s", *bin.Script))
	}
	cmd := fmt.Sprintf("%s %s -o %s %s", toolchain.Cxx, strings.Join(flags, " "), bin.Out, objs)
	ldStep := core.BuildStep{
		Out:   bin.Out,
		Ins:   objs,
		Cmd:   cmd,
		Descr: fmt.Sprintf("LD %s", bin.Out.RelPath()),
		Alias: bin.Out.RelPath(),
	}

	return append(steps, ldStep)
}
