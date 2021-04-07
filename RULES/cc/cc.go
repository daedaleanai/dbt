package cc

import (
	"fmt"

	"dbt/RULES/core"
)

// ObjectFile compiles a single C++ source file.
type ObjectFile struct {
	Src       core.Path
	Includes  []core.Path
	Flags     []string
	Toolchain Toolchain
}

// Build an ObjectFile.
func (obj ObjectFile) Build(ctx core.Context) {
	toolchain := obj.Toolchain
	if toolchain == nil {
		toolchain = &defaultToolchain
	}

	depfile := obj.Src.WithExt("d")
	cmd := toolchain.ObjectFile(obj.Output(), depfile, obj.Flags, obj.Includes, obj.Src)
	ctx.AddBuildStep(core.BuildStep{
		Out:     obj.Output(),
		Depfile: depfile,
		In:      obj.Src,
		Cmd:     cmd,
		Descr:   fmt.Sprintf("CC %s", obj.Output().Relative()),
	})
}

// Output for ObjectFile returns the produced object file.
func (obj ObjectFile) Output() core.OutPath {
	return obj.Src.WithExt("o")
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

func compileSources(ctx core.Context, srcs []core.Path, flags []string, deps []Library, toolchain Toolchain) []core.Path {
	includes := []core.Path{core.NewInPath(".")}
	for _, dep := range deps {
		includes = append(includes, dep.Includes...)
	}

	objs := []core.Path{}

	for _, src := range srcs {
		obj := ObjectFile{
			Src:       src,
			Includes:  includes,
			Flags:     flags,
			Toolchain: toolchain,
		}
		obj.Build(ctx)
		objs = append(objs, obj.Output())
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
	Srcs          []core.Path
	Objs          []core.Path
	Includes      []core.Path
	CompilerFlags []string
	Deps          []Dep
	Shared        bool
	AlwaysLink    bool
	Toolchain     Toolchain
}

// Build a Library.
func (lib Library) Build(ctx core.Context) {
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
}

// Output for Library returns the produced library.
func (lib Library) Output() core.OutPath {
	return lib.Out
}

// CcLibrary for Library is just the identity.
func (lib Library) CcLibrary() Library {
	return lib
}

// Binary builds and links an executable.
type Binary struct {
	Out           core.OutPath
	Srcs          []core.Path
	CompilerFlags []string
	LinkerFlags   []string
	Deps          []Dep
	Script        core.Path
	Toolchain     Toolchain
}

// Build a Binary.
func (bin Binary) Build(ctx core.Context) {
	toolchain := bin.Toolchain
	if toolchain == nil {
		toolchain = defaultToolchain
	}

	deps := flattenDeps(bin.Deps)
	objs := compileSources(ctx, bin.Srcs, bin.CompilerFlags, deps, toolchain)

	ins := objs
	alwaysLinkLibs := []core.Path{}
	otherLibs := []core.Path{}
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
}

// Output for Binary returns the resulting binary.
func (bin Binary) Output() core.OutPath {
	return bin.Out
}
