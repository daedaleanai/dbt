package cc

import (
	"fmt"
	"strings"

	"dbt/RULES/core"
)

type Toolchain interface {
	ObjectFile(out core.OutPath, depfile core.OutPath, flags core.Flags, includes core.Paths, src core.Path) string
	Library(out core.Path, objs core.Paths) string
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
		includesStr.WriteString(fmt.Sprintf("-I%s ", include))
	}
	for _, include := range gcc.Includes {
		includesStr.WriteString(fmt.Sprintf("-isystem %s ", include))
	}

	return fmt.Sprintf(
		"%s -c -o %s -MD -MF %s %s %s %s",
		gcc.Cxx,
		out,
		depfile,
		append(gcc.CompilerFlags, flags...),
		includesStr.String(),
		src)
}

// Library generates the command to build a static library.
func (gcc GccToolchain) Library(out core.Path, objs core.Paths) string {
	return fmt.Sprintf(
		"%s rv %s %s >/dev/null 2>/dev/null",
		gcc.Ar,
		out,
		objs)
}

// Binary generates the command to build an executable.
func (gcc GccToolchain) Binary(out core.Path, objs core.Paths, alwaysLinkLibs core.Paths, libs core.Paths, flags core.Flags, script core.Path) string {
	flags = append(gcc.LinkerFlags, flags...)
	if script != nil {
		flags = append(flags, "-T", script.String())
	}

	return fmt.Sprintf(
		"%s -o %s %s -Wl,-whole-archive %s -Wl,-no-whole-archive %s %s",
		gcc.Cxx,
		out,
		objs,
		alwaysLinkLibs,
		libs,
		flags)
}

func (gcc GccToolchain) EmbeddedBlob(out core.OutPath, src core.Path) string {
	return fmt.Sprintf(
		"%s -I binary -O %s -B %s %s %s",
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
