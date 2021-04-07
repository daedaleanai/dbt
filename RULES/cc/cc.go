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

// Build an ObjectFile.
func (obj ObjectFile) Build(ctx core.Context) core.OutPath {
	toolchain := obj.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	out := obj.Src.WithExt("o")
	depfile := obj.Src.WithExt("d")
	cmd := toolchain.ObjectFile(out, depfile, obj.Flags, obj.Includes, obj.Src)
	ctx.AddBuildStep(core.BuildStep{
		Out:     out,
		Depfile: depfile,
		In:      obj.Src,
		Cmd:     cmd,
		Descr:   fmt.Sprintf("CC %s", out.Relative()),
	})

	return out
}

func flattenDepsRec(deps []Dep, visited map[string]bool) []Library {
	flatDeps := []Library{}
	for _, dep := range deps {
		lib := dep.CcLibrary()
		libPath := lib.Out.Absolute()
		if _, exists := visited[libPath]; !exists {
			visited[libPath] = true
			flatDeps = append([]Library{lib}, append(flattenDepsRec(lib.Deps, visited), flatDeps...)...)
		}
	}
	return flatDeps
}

func flattenDeps(deps []Dep) []Library {
	return flattenDepsRec(deps, map[string]bool{})
}

func compileSources(ctx core.Context, srcs core.Paths, flags core.Flags, deps []Library, toolchain Toolchain) core.Paths {
	includes := core.Paths{core.NewInPath(".")}
	for _, dep := range deps {
		includes = append(includes, dep.Includes...)
	}

	objs := core.Paths{}

	for _, src := range srcs {
		obj := ObjectFile{
			Src:       src,
			Includes:  includes,
			Flags:     flags,
			Toolchain: toolchain,
		}
		out := obj.Build(ctx)
		objs = append(objs, out)
	}

	return objs
}

// Dep is an interface implemented by dependencies that can be linked into a library.
type Dep interface {
	CcLibrary() Library
}

// Library builds and links a static C++ library.
type Library struct {
	Out           core.OutPath
	Srcs          core.Paths
	Objs          core.Paths
	Includes      core.Paths
	CompilerFlags core.Flags
	Deps          []Dep
	Shared        bool
	AlwaysLink    bool
	Toolchain     Toolchain
}

// Build a Library.
func (lib Library) Build(ctx core.Context) core.OutPath {
	toolchain := lib.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	objs := compileSources(ctx, lib.Srcs, lib.CompilerFlags, flattenDeps([]Dep{lib}), toolchain)
	objs = append(objs, lib.Objs...)

	var cmd, descr string
	if lib.Shared {
		cmd = toolchain.SharedLibrary(lib.Out, objs)
		descr = fmt.Sprintf("LD %s", lib.Out.Relative())
	} else {
		cmd = toolchain.StaticLibrary(lib.Out, objs)
		descr = fmt.Sprintf("AR %s", lib.Out.Relative())
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:   lib.Out,
		Ins:   objs,
		Cmd:   cmd,
		Descr: descr,
	})

	return lib.Out
}

// CcLibrary for Library is just the identity.
func (lib Library) CcLibrary() Library {
	return lib
}

// Binary builds and links an executable.
type Binary struct {
	Out           core.OutPath
	Srcs          core.Paths
	CompilerFlags core.Flags
	LinkerFlags   core.Flags
	Deps          []Dep
	Script        core.Path
	Toolchain     Toolchain
}

// Build a Binary.
func (bin Binary) Build(ctx core.Context) core.OutPath {
	toolchain := bin.Toolchain
	if toolchain == nil {
		toolchain = defaultToolchain
	}

	deps := flattenDeps(bin.Deps)
	objs := compileSources(ctx, bin.Srcs, bin.CompilerFlags, deps, toolchain)

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
	ctx.AddBuildStep(core.BuildStep{
		Out:   bin.Out,
		Ins:   ins,
		Cmd:   cmd,
		Descr: fmt.Sprintf("LD %s", bin.Out.Relative()),
	})

	return bin.Out
}
