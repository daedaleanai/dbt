package cc

import (
	"fmt"
	"strings"

	"dbt/RULES/core"
)

// Toolchain represents a C++ toolchain.
type Toolchain struct {
	Ar      core.GlobalPath
	As      core.GlobalPath
	Cc      core.GlobalPath
	Cpp     core.GlobalPath
	Cxx     core.GlobalPath
	Objcopy core.GlobalPath

	Includes core.Paths

	CompilerFlags core.Flags
	LinkerFlags   core.Flags

	CrtBegin core.Path
	CrtEnd   core.Path
}

var defaultToolchain = Toolchain{
	Ar:      core.NewGlobalPath("ar"),
	As:      core.NewGlobalPath("as"),
	Cc:      core.NewGlobalPath("gcc"),
	Cpp:     core.NewGlobalPath("gcc -E"),
	Cxx:     core.NewGlobalPath("g++"),
	Objcopy: core.NewGlobalPath("objcopy"),

	CompilerFlags: core.Flags{"-std=c++14", "-O3", "-fdiagnostics-color=always"},
	LinkerFlags:   core.Flags{"-fdiagnostics-color=always"},
}

// ObjectFile compiles a single C++ source file.
type ObjectFile struct {
	Src       core.Path
	Includes  core.Paths
	Flags     core.Flags
	Toolchain *Toolchain
}

// Out provides the name of the output created by ObjectFile.
func (obj ObjectFile) Out() core.OutPath {
	return obj.Src.WithExt("o")
}

// BuildSteps for ObjectFile.
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
		Descr:   fmt.Sprintf("CC %s", obj.Out().Relative()),
		Alias:   obj.Out().Relative(),
	}}
}

func flattenDeps(deps []Library) []Library {
	allDeps := append([]Library{}, deps...)
	for _, dep := range deps {
		allDeps = append(allDeps, flattenDeps(dep.Deps)...)
	}
	return allDeps
}

func compileSources(srcs core.Paths, flags core.Flags, deps []Library, toolchain *Toolchain) ([]core.BuildStep, core.Paths) {
	includes := core.Paths{core.NewInPath(".")}
	for _, dep := range deps {
		includes = append(includes, dep.Includes...)
	}

	steps := []core.BuildStep{}
	objs := core.Paths{}

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

// Library builds and links a static C++ library.
type Library struct {
	Out           core.OutPath
	Srcs          core.Paths
	Objs          core.Paths
	Includes      core.Paths
	CompilerFlags core.Flags
	Deps          []Library
	AlwaysLink    bool
	Toolchain     *Toolchain
}

// BuildSteps for Library.
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
		Descr: fmt.Sprintf("AR %s", lib.Out.Relative()),
		Alias: lib.Out.Relative(),
	}

	return append(steps, arStep)
}

// Binary builds and links an executable.
type Binary struct {
	Out           core.OutPath
	Srcs          core.Paths
	CompilerFlags core.Flags
	LinkerFlags   core.Flags
	Deps          []Library
	Script        core.Path
	Toolchain     *Toolchain
}

// BuildSteps for Binary.
func (bin Binary) BuildSteps() []core.BuildStep {
	toolchain := bin.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	deps := flattenDeps(bin.Deps)
	steps, objs := compileSources(bin.Srcs, bin.CompilerFlags, deps, toolchain)

	ins := objs
	seenLibs := map[string]struct{}{}
	alwaysLinkLibs := core.Paths{}
	otherLibs := core.Paths{}
	for _, dep := range deps {
		if _, exists := seenLibs[dep.Out.Absolute()]; !exists {
			ins = append(ins, dep.Out)
			seenLibs[dep.Out.Absolute()] = struct{}{}
			if dep.AlwaysLink {
				alwaysLinkLibs = append(alwaysLinkLibs, dep.Out)
			} else {
				otherLibs = append(otherLibs, dep.Out)
			}
		}
	}

	flags := append(toolchain.LinkerFlags, bin.LinkerFlags...)
	if bin.Script != nil {
		flags = append(flags, "-T", bin.Script.String())
		ins = append(ins, bin.Script)
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
		Descr: fmt.Sprintf("LD %s", bin.Out.Relative()),
		Alias: bin.Out.Relative(),
	}

	return append(steps, ldStep)
}
