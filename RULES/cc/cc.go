package cc

import (
	"fmt"

	"dbt/RULES/core"
)

// ObjectFile compiles a single C++ source file.
type ObjectFile struct {
	Src       core.Path
	Includes  core.Paths
	Flags     core.Flags
	Toolchain Toolchain
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

	depfile := obj.Src.WithExt("d")
	cmd := toolchain.ObjectFile(obj.Out(), depfile, obj.Flags, obj.Includes, obj.Src)
	return []core.BuildStep{{
		Out:     obj.Out(),
		Depfile: &depfile,
		In:      obj.Src,
		Cmd:     cmd,
		Descr:   fmt.Sprintf("CC %s", obj.Out().Relative()),
		Alias:   obj.Out().Relative(),
	}}
}

func flattenDepsRec(deps []Library, visited map[string]bool) []Library {
	flatDeps := []Library{}
	for _, dep := range deps {
		libPath := dep.Out.Absolute()
		if _, exists := visited[libPath]; !exists {
			visited[libPath] = true
			flatDeps = append([]Library{dep}, append(flattenDepsRec(dep.Deps, visited), flatDeps...)...)
		}
	}
	return flatDeps
}

func flattenDeps(deps []Library) []Library {
	return flattenDepsRec(deps, map[string]bool{})
}

func compileSources(srcs core.Paths, flags core.Flags, deps []Library, toolchain Toolchain) ([]core.BuildStep, core.Paths) {
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
	Toolchain     Toolchain
}

// BuildSteps for Library.
func (lib Library) BuildSteps() []core.BuildStep {
	toolchain := lib.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	steps, objs := compileSources(lib.Srcs, lib.CompilerFlags, flattenDeps([]Library{lib}), toolchain)
	objs = append(objs, lib.Objs...)

	cmd := toolchain.Library(lib.Out, objs)
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
	Toolchain     Toolchain
}

// BuildSteps for Binary.
func (bin Binary) BuildSteps() []core.BuildStep {
	toolchain := bin.Toolchain
	if toolchain == nil {
		toolchain = defaultToolchain
	}

	deps := flattenDeps(bin.Deps)
	steps, objs := compileSources(bin.Srcs, bin.CompilerFlags, deps, toolchain)

	ins := objs
	alwaysLinkLibs := core.Paths{}
	otherLibs := core.Paths{}
	for _, dep := range deps {
		ins = append(ins, dep.Out)
		if dep.AlwaysLink {
			alwaysLinkLibs = append(alwaysLinkLibs, dep.Out)
		} else {
			otherLibs = append(otherLibs, dep.Out)
		}
	}

	if bin.Script != nil {
		ins = append(ins, bin.Script)
	}

	cmd := toolchain.Binary(bin.Out, objs, alwaysLinkLibs, otherLibs, bin.LinkerFlags, bin.Script)
	ldStep := core.BuildStep{
		Out:   bin.Out,
		Ins:   ins,
		Cmd:   cmd,
		Descr: fmt.Sprintf("LD %s", bin.Out.Relative()),
		Alias: bin.Out.Relative(),
	}

	return append(steps, ldStep)
}
