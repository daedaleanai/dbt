package cc

import (
	"fmt"
	"strings"

	"dbt/RULES/core"
)

type Toolchain interface {
	ObjectFile(out core.OutPath, depfile core.OutPath, flags core.Flags, includes core.Paths, src core.Path) string
	StaticLibrary(out core.Path, objs core.Paths) string
	SharedLibrary(out core.Path, objs core.Paths) string
	Binary(out core.Path, objs core.Paths, alwaysLinkLibs core.Paths, libs core.Paths, flags core.Flags, script core.Path) string
	EmbeddedBlob(out core.OutPath, src core.Path) string
}

// Toolchain represents a C++ toolchain.
type GccToolchain struct {
	Ar      core.GlobalPath
	As      core.GlobalPath
	Cc      core.GlobalPath
	Cpp     core.GlobalPath
	Cxx     core.GlobalPath
	Objcopy core.GlobalPath

	Includes core.Paths

	CompilerFlags core.Flags
	LinkerFlags   core.Flags

	ArchName   string
	TargetName string
}

// ObjectFile generates a compile command.
func (gcc GccToolchain) ObjectFile(out core.OutPath, depfile core.OutPath, flags core.Flags, includes core.Paths, src core.Path) string {
	includesStr := strings.Builder{}
	for _, include := range includes {
		includesStr.WriteString(fmt.Sprintf("-I%q ", include))
	}
	for _, include := range gcc.Includes {
		includesStr.WriteString(fmt.Sprintf("-isystem %q ", include))
	}

	return fmt.Sprintf(
		"%q -pipe -c -o %q -MD -MF %q %s %s %q",
		gcc.Cxx,
		out,
		depfile,
		append(gcc.CompilerFlags, flags...),
		includesStr.String(),
		src)
}

// StaticLibrary generates the command to build a static library.
func (gcc GccToolchain) StaticLibrary(out core.Path, objs core.Paths) string {
	return fmt.Sprintf(
		"%q rv %q %s >/dev/null 2>/dev/null",
		gcc.Ar,
		out,
		objs)
}

// SharedLibrary generates the command to build a shared library.
func (gcc GccToolchain) SharedLibrary(out core.Path, objs core.Paths) string {
	return fmt.Sprintf(
		"%q -pipe -shared -o %q %s",
		gcc.Cxx,
		out,
		objs)
}

// Binary generates the command to build an executable.
func (gcc GccToolchain) Binary(out core.Path, objs core.Paths, alwaysLinkLibs core.Paths, libs core.Paths, flags core.Flags, script core.Path) string {
	flags = append(gcc.LinkerFlags, flags...)
	if script != nil {
		flags = append(flags, "-T", fmt.Sprintf("%q", script))
	}

	return fmt.Sprintf(
		"%q -pipe -o %q %s -Wl,-whole-archive %s -Wl,-no-whole-archive %s %s",
		gcc.Cxx,
		out,
		objs,
		alwaysLinkLibs,
		libs,
		flags)
}

func (gcc GccToolchain) EmbeddedBlob(out core.OutPath, src core.Path) string {
	return fmt.Sprintf(
		"%q -I binary -O %s -B %s %q %q",
		gcc.Objcopy,
		gcc.TargetName,
		gcc.ArchName,
		src,
		out)
}

var defaultToolchain = GccToolchain{
	Ar:      core.NewGlobalPath("ar"),
	As:      core.NewGlobalPath("as"),
	Cc:      core.NewGlobalPath("gcc"),
	Cpp:     core.NewGlobalPath("gcc -E"),
	Cxx:     core.NewGlobalPath("g++"),
	Objcopy: core.NewGlobalPath("objcopy"),

	CompilerFlags: core.Flags{"-std=c++14", "-O3", "-fdiagnostics-color=always"},
	LinkerFlags:   core.Flags{"-fdiagnostics-color=always"},
}
