package cc

import (
	"fmt"
	"sort"
	"strings"

	"dbt-rules/RULES/core"
)

type LinkerFlavor int

const (
	Ld LinkerFlavor = iota
	LdLld
	LldLink
	Gcc
	Clang
)

type Toolchain interface {
	Name() string

	CCompiler() string
	CxxCompiler() string
	Assembler() string
	Archiver() string
	Link() string
	ObjcopyCommand() string

	CFlags() []string
	CxxFlags() []string
	AsFlags() []string

	LdFlags() []string

	StdDeps() []Dep
	Script() core.Path

	LinkerFlavor() LinkerFlavor
}

type Architecture string

const (
	ArchitectureX86_64  Architecture = "x86_64"
	ArchitectureAArch64 Architecture = "aarch64"
	ArchitectureArmv6m  Architecture = "armv6m"
	ArchitectureArmv7m  Architecture = "armv7m"
	ArchitectureArmv7em Architecture = "armv7em"
	ArchitectureArmv8m  Architecture = "armv8m"
	ArchitectureUnknown Architecture = "Unknown"
)

// ToolchainArchitecture returns the architecture for the toolchain if known.
func ToolchainArchitecture(toolchain Toolchain) Architecture {
	if tca, ok := toolchain.(interface{ Architecture() Architecture }); ok {
		return tca.Architecture()
	}
	return ArchitectureUnknown
}

// ToolchainFreestanding reports whether the toolchain uses a
// freestanding environment (rather than a hosted one).
func ToolchainFreestanding(toolchain Toolchain) bool {
	if tcf, ok := toolchain.(interface{ Freestanding() bool }); ok {
		return tcf.Freestanding()
	}
	return false
}

// Toolchain represents a C++ toolchain.
type GccToolchain struct {
	Ar      core.GlobalPath
	As      core.GlobalPath
	Cc      core.GlobalPath
	Cpp     core.GlobalPath
	Cxx     core.GlobalPath
	Objcopy core.GlobalPath
	Ld      core.GlobalPath

	Includes     []core.Path
	Deps         []Dep
	LinkerScript core.Path

	CCompilerFlags   []string
	CxxCompilerFlags []string
	AsCompilerFlags  []string
	LinkerFlags      []string

	ToolchainName string
	ArchName      string
	TargetName    string
}

func (gcc GccToolchain) Architecture() Architecture {
	// TODO: remove i386, which appears to be a typo in libsupcxx
	if gcc.ArchName == "i386" || gcc.ArchName == "x86_64" {
		return ArchitectureX86_64
	}
	if gcc.ArchName == "aarch64" {
		return ArchitectureAArch64
	}
	return ArchitectureUnknown
}

func (gcc GccToolchain) Freestanding() bool {
	for _, lf := range gcc.LinkerFlags {
		if lf == "-ffreestanding" {
			return true
		}
	}
	return false
}

func (gcc GccToolchain) CCompiler() string {
	return fmt.Sprintf("%q", gcc.Cc)
}

func (gcc GccToolchain) CxxCompiler() string {
	return fmt.Sprintf("%q", gcc.Cxx)
}

func (gcc GccToolchain) Link() string {
	return fmt.Sprintf("%q", gcc.Ld)
}

func (gcc GccToolchain) Assembler() string {
	return fmt.Sprintf("%q", gcc.As)
}

func (gcc GccToolchain) Archiver() string {
	return fmt.Sprintf("%q", gcc.Ar)
}

func (gcc GccToolchain) ObjcopyCommand() string {
	return fmt.Sprintf("%q", gcc.Objcopy)
}

func (gcc GccToolchain) CFlags() []string {
	result := gcc.CCompilerFlags
	for _, inc := range gcc.Includes {
		result = append(result, "-isystem", fmt.Sprintf("%q", inc))
	}
	return result
}

func (gcc GccToolchain) CxxFlags() []string {
	result := gcc.CxxCompilerFlags
	for _, inc := range gcc.Includes {
		result = append(result, "-isystem", fmt.Sprintf("%q", inc))
	}
	return result
}

func (gcc GccToolchain) AsFlags() []string {
	return gcc.AsCompilerFlags
}

func (gcc GccToolchain) LdFlags() []string {
	return gcc.LinkerFlags
}

func (gcc GccToolchain) NewWithStdLib(includes []core.Path, deps []Dep, linkerScript core.Path, toolchainName string) GccToolchain {
	gcc.Includes = includes
	gcc.Deps = deps
	gcc.LinkerScript = linkerScript
	gcc.ToolchainName = toolchainName
	return gcc
}

func (gcc GccToolchain) StdDeps() []Dep {
	return gcc.Deps
}

func (gcc GccToolchain) Script() core.Path {
	return gcc.LinkerScript
}
func (gcc GccToolchain) Name() string {
	return gcc.ToolchainName
}

func (gcc GccToolchain) LinkerFlavor() LinkerFlavor {
	return Ld
}

func joinQuoted(paths []core.Path) string {
	b := strings.Builder{}
	for _, p := range paths {
		fmt.Fprintf(&b, "%q ", p)
	}
	return b.String()
}

var toolchains = make(map[string]Toolchain)

func RegisterToolchain(toolchain Toolchain) Toolchain {
	if _, found := toolchains[toolchain.Name()]; found {
		core.Fatal("A toolchain with name %s has already been registered", toolchain.Name())
	}
	toolchains[toolchain.Name()] = toolchain
	return toolchain
}

var defaultToolchainFlag = core.StringFlag{
	Name:        "cc-toolchain",
	Description: "Overrides the default toolchain to compile generic C/C++ targets",
	DefaultFn:   func() string { return "invalid-toolchain" },
}.Register()

var defaultToolchain = ""

// Registers a toolchain as the default toolchain.
func RegisterToolchainAsDefault(toolchain Toolchain) Toolchain {
	if defaultToolchain != "" {
		core.Fatal("Default toolchain is already registered to ", defaultToolchain, ", but attempted to register: ", toolchain.Name())
	}
	defaultToolchain = toolchain.Name()
	return RegisterToolchain(toolchain)
}

// DefaultToolchain returns the default toolchain: either the registered default toolchain
// (via RegisterToolchainAsDefault) or the overriden default toolchain specified on the
// command-line with the cc-toolchain flag.
func DefaultToolchain() Toolchain {
	if defaultToolchainFlag.Value() != "invalid-toolchain" {
		if toolchain, ok := toolchains[defaultToolchainFlag.Value()]; ok {
			return toolchain
		}
	} else if toolchain, ok := toolchains[defaultToolchain]; ok {
		return toolchain
	}

	var all []string
	for tc, _ := range toolchains {
		all = append(all, fmt.Sprintf("%q", tc))
	}
	sort.Strings(all)
	core.Fatal("No registered toolchain %q. Registered toolchains: %s", defaultToolchainFlag.Value(), strings.Join(all, ", "))
	return nil
}

func toolchainOrDefault(toolchain Toolchain) Toolchain {
	if toolchain == nil {
		return DefaultToolchain()
	}
	return toolchain
}
