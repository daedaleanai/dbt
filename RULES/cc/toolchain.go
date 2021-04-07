package cc

import (
	"fmt"
	"strings"

	"dbt/RULES/core"
)

type Toolchain interface {
	ObjectFile(out core.OutPath, depfile core.OutPath, flags []string, includes []core.Path, src core.Path) string
	StaticLibrary(out core.Path, objs []core.Path) string
	SharedLibrary(out core.Path, objs []core.Path) string
	Binary(out core.Path, objs []core.Path, alwaysLinkLibs []core.Path, libs []core.Path, flags []string, script core.Path) string
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

	Includes []core.Path

	CompilerFlags []string
	LinkerFlags   []string

	ArchName   string
	TargetName string
}

// ObjectFile generates a compile command.
func (gcc GccToolchain) ObjectFile(out core.OutPath, depfile core.OutPath, flags []string, includes []core.Path, src core.Path) string {
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
		strings.Join(append(gcc.CompilerFlags, flags...), " "),
		includesStr.String(),
		src)
}

// StaticLibrary generates the command to build a static library.
func (gcc GccToolchain) StaticLibrary(out core.Path, objs []core.Path) string {
	return fmt.Sprintf(
		"%q rv %q %s >/dev/null 2>/dev/null",
		gcc.Ar,
		out,
		joinQuoted(objs))
}

// SharedLibrary generates the command to build a shared library.
func (gcc GccToolchain) SharedLibrary(out core.Path, objs []core.Path) string {
	return fmt.Sprintf(
		"%q -pipe -shared -o %q %s",
		gcc.Cxx,
		out,
		joinQuoted(objs))
}

// Binary generates the command to build an executable.
func (gcc GccToolchain) Binary(out core.Path, objs []core.Path, alwaysLinkLibs []core.Path, libs []core.Path, flags []string, script core.Path) string {
	flags = append(gcc.LinkerFlags, flags...)
	if script != nil {
		flags = append(flags, "-T", fmt.Sprintf("%q", script))
	}

	return fmt.Sprintf(
		"%q -pipe -o %q %s -Wl,-whole-archive %s -Wl,-no-whole-archive %s %s",
		gcc.Cxx,
		out,
		joinQuoted(objs),
		joinQuoted(alwaysLinkLibs),
		joinQuoted(libs),
		strings.Join(flags, " "))
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

func joinQuoted(paths []core.Path) string {
	b := strings.Builder{}
	for _, p := range paths {
		fmt.Fprintf(&b, "%q ", p)
	}
	return b.String()
}

var defaultToolchain = GccToolchain{
	Ar:      core.NewGlobalPath("ar"),
	As:      core.NewGlobalPath("as"),
	Cc:      core.NewGlobalPath("gcc"),
	Cpp:     core.NewGlobalPath("gcc -E"),
	Cxx:     core.NewGlobalPath("g++"),
	Objcopy: core.NewGlobalPath("objcopy"),

	CompilerFlags: []string{"-std=c++14", "-O3", "-fdiagnostics-color=always"},
	LinkerFlags:   []string{"-fdiagnostics-color=always"},
}
