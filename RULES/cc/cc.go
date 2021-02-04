package cc

import (
	"fmt"
	"strings"

	"dbt/RULES/core"
)

type Flags []string

func (f Flags) String() string {
	return strings.Join(f, " ")
}

// Toolchain represents a C++ toolchain.
type Toolchain struct {
	Ar      core.GlobalFile
	As      core.GlobalFile
	Cc      core.GlobalFile
	Cpp     core.GlobalFile
	Cxx     core.GlobalFile
	Objcopy core.GlobalFile

	Includes core.Files

	CompilerFlags Flags
	LinkerFlags   Flags

	CrtBegin core.File
	CrtEnd   core.File
}

var defaultToolchain = Toolchain{
	Ar:      core.NewGlobalFile("ar"),
	As:      core.NewGlobalFile("as"),
	Cc:      core.NewGlobalFile("gcc"),
	Cpp:     core.NewGlobalFile("g++"),
	Cxx:     core.NewGlobalFile("gcc"),
	Objcopy: core.NewGlobalFile("objcopy"),

	CompilerFlags: Flags{"-std=c++14", "-O3", "-fdiagnostics-color=always"},
	LinkerFlags:   Flags{"-fdiagnostics-color=always"},
}

// ObjectFile compiles a single C++ source file.
type ObjectFile struct {
	Src       core.File
	Includes  core.Files
	Flags     Flags
	Toolchain *Toolchain
}

// Out provides the name of the output created by ObjectFile.
func (obj ObjectFile) Out() core.OutFile {
	return obj.Src.WithExt("o")
}

// BuildSteps provides the steps to produce an ObjectFile.
func (obj ObjectFile) BuildSteps() []core.BuildStep {
	toolchain := obj.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	includes := strings.Builder{}
	for _, include := range obj.Includes {
		includes.WriteString(fmt.Sprintf("-I%s ", include))
	}
	for _, include := range toolchain.Includes {
		includes.WriteString(fmt.Sprintf("-isystem %s ", include))
	}
	flags := append(toolchain.CompilerFlags, obj.Flags...)
	depfile := obj.Src.WithExt("d")
	cmd := fmt.Sprintf(
		"%s -c -o %s -MD -MF %s %s %s %s",
		obj.Toolchain.Cxx,
		obj.Out(),
		depfile,
		flags,
		includes.String(),
		obj.Src)
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

func compileSources(srcs core.Files, flags Flags, deps []Library, toolchain *Toolchain) ([]core.BuildStep, core.Files) {
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
	Out           core.OutFile
	Srcs          core.Files
	Objs          core.Files
	Includes      core.Files
	CompilerFlags Flags
	Deps          []Library
	AlwaysLink    bool
	Toolchain     *Toolchain
}

// BuildSteps provides the steps to build a Library.
func (lib Library) BuildSteps() []core.BuildStep {
	toolchain := lib.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	steps, objs := compileSources(lib.Srcs, lib.CompilerFlags, flattenDeps([]Library{lib}), toolchain)
	objs = append(objs, lib.Objs...)

	cmd := fmt.Sprintf("%s rv %s %s >/dev/null 2>/dev/null", toolchain.Ar, lib.Out, objs)
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
	Out           core.OutFile
	Srcs          core.Files
	CompilerFlags Flags
	LinkerFlags   Flags
	Deps          []Library
	Script        *core.File
	Toolchain     *Toolchain
}

// BuildSteps provides the steps to build a Binary.
func (bin Binary) BuildSteps() []core.BuildStep {
	toolchain := bin.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	deps := flattenDeps(bin.Deps)
	steps, objs := compileSources(bin.Srcs, bin.CompilerFlags, deps, toolchain)

	ins := objs
	seenLibs := map[string]struct{}{}
	alwaysLinkLibs := core.Files{}
	otherLibs := core.Files{}
	for _, dep := range deps {
		if _, exists := seenLibs[dep.Out.Path()]; !exists {
			ins = append(ins, dep.Out)
			seenLibs[dep.Out.Path()] = struct{}{}
			if dep.AlwaysLink {
				alwaysLinkLibs = append(alwaysLinkLibs, dep.Out)
			} else {
				otherLibs = append(otherLibs, dep.Out)
			}
		}
	}

	flags := append(toolchain.LinkerFlags, bin.LinkerFlags...)
	if bin.Script != nil {
		flags = append(flags, "-T", (*bin.Script).String())
		ins = append(ins, *bin.Script)
	}
	cmd := fmt.Sprintf(
		"%s -o %s %s -Wl,-whole-archive %s -Wl,-no-whole-archive %s %s",
		toolchain.Cxx,
		bin.Out,
		objs,
		alwaysLinkLibs,
		otherLibs,
		flags)
	ldStep := core.BuildStep{
		Out:   bin.Out,
		Ins:   ins,
		Cmd:   cmd,
		Descr: fmt.Sprintf("LD %s", bin.Out.RelPath()),
		Alias: bin.Out.RelPath(),
	}

	return append(steps, ldStep)
}
