# Daedalean Build Tool

The Daedalean Build Tool (DBT) provides dependency management and build system features in a single tool. The dependency management allows setting up workspaces consisting of multiple modules and managing dependencies between different modules. The build system components allows defining build targets and custom build rules in Go.

## Requirements

DBT requires the following tools to be installed on your system:

* go (>= 1.18)
* git
* ninja

## Installation

You can install the latest version of DBT by running:

```
go install github.com/daedaleanai/dbt/v3@latest
```

Notice that DBT follows sematic versioning, and major version change introduce breaking changes.
You might in practice prefer to pin to a specific major version.

CHANGELOG.md lists the changes between versions.

### Setting up a local mirror

DBT can read a configuration file located at:
- `$DBT_CONFIG_DIR/config.yaml`, if `$DBT_CONFIG_DIR` is set.
- `$XDG_CONFIG_DIR/dbt/config.yaml`, if `$XDG_CONFIG_DIR` is set.
- `$HOME/.config/dbt/config.yaml`, if `$HOME` is set.
- Otherwise no configuration is loaded.

This configuration file allows the user to configure a DBT local mirror with the following line:

```yaml
mirror: "<PATH_TO_YOUR_LOCAL_MIRROR>"
```

Replace `<PATH_TO_YOUR_LOCAL_MIRROR>` by a full path in your system where your user has read and
write permissions. Note that this path MUST exist for the mirror to be used by DBT.

With a local mirror configured, DBT will reduce the amount of bandwidth required to sync dependencies.
In particular, its behavior is different between archives and git repositories:
- Compressed archives (`*.tar.gz`): they get downloaded first into the local mirror and then
copied to your project's dependency folder. If they are already available in your local mirror, they
are simply copied over to your dependency folder, so no network access is required.
- Git repositories: they get cloned with the `--mirror` flag in the mirror directory. In your dependency
folder, they get cloned using the url and a `--reference` flag pointing to the local mirror. For them,
some network access might be required (e.g., if the branch has updates), but the bulk of fetching
all git objects can be done from the local mirror.

Note that the data in the mirror is never deleted/freed by DBT. It is the user's responsibility 
to manage it and delete old checkouts that are not required anymore when disk usage gets too large.

## General remarks

* All DBT commands have a `-v` / `--verbose` flag to enable debug output.
* `dbt --version` prints the current version of the tool.
* DBT supports shell completion for `bash`, `zsh`, and `fish` shells. Run `dbt completion bash|zsh|fish` to get the respective completion script.
* The auto-generated Go documentation for this repository can be found [here](https://pkg.go.dev/github.com/daedaleanai/dbt).
* The auto-generated Go documentation for the `dbt-rules` repository can be found [here](https://pkg.go.dev/github.com/daedaleanai/dbt-rules).

## Dependency management

In DBT, dependency management is centered around the concept of modules. DBT currently supports two types of modules: Git repositories and `.tar.gz` archives.

Each module contains a `MODULE` file in its root directory to declare its dependencies on other modules. Modules always depend on a _named version_ of another module. In case of a Git dependency, this can be a branch name, tag or commit hash. `.tar.gz` archive dependencies only have a single version called `master`. When depending on a Git branch, the dependency should be against the remote branch (e.g., `origin/some-banch`) to ensure updates to the branch are considered by DBT.

When a dependency is pinned for the first time (i.e., when running the `dbt sync` command), the dependency version (as specified in the `MODULE` file of the dependent module) is resolved to a hash that uniquely identifies a snapshot of the dependency. For Git dependencies this is the commit hash, for `.tar.gz` archives this is the `sha256` hash of the archives content.

The resolved hash is then added to the `MODULE` file of the dependent module. To guarantee reproducible builds, DBT will always use the hash from the `MODULE` file to resolve a dependency, if it is available. In order to update these hashes (e.g., when a dependency on a Git branch should reflect new commits), use `dbt sync ---update`.

## Directory structure

There is no explicit concept of workspaces. Instead, each module can "become" a workspace when running the `dbt sync` command in the module directory. This module is then called the top-level module or workspace. The `dbt sync` command creates a `DEPS/` directory in the workspace's root directory. All direct and transitive dependencies will be stored inside the `DEPS/` directory. Furthermore, a symlink from the workspace root directory into the `DEPS/` directory is created. The symlink ensures that all modules can access their dependencies as sibling directories regardles of which module acts as the workspace.

### Manipulating MODULE files

`MODULE` files should rarely (if ever) be edited by hand. Instead, the following commands should be used to add, remove and update dependencies.
The following commands always act on the `MODULE` file of the current module (according to the working directory) and not necessarily the top-level module. The commands only edit `MODULE` files but will never download, delete, clone or in any other way change any other than the current module.

#### Adding a dependency

To add a dependency to the current module run:
```
dbt dep add [NAME] --url=URL --version=VERSION
```

The `NAME` parameter determines the name of the module directory inside the `DEPS/` directory. It is derived from the `URL` if omitted.
In order to change the version of the dependency (e.g., to depend on another version of a dependency), simply rerun the `dbt dep add` command.

#### Removing a dependency

To remove a dependency from the current module run:
```
dbt dep remove NAME
```

### Module initialization

If a module has a `SETUP.go` file in its root directory, DBT will run the `SETUP.go` whenever a new snapshot of the module is checked out. This mechanism can be be used to initialize modules (e.g. install git hooks). The `SETUP.go` scripts should thus be written in an idempotent way. DBT enforces a 10 second time limit on `SETUP.go` scripts.

### The sync command

The `dbt sync [--update] [--ignore-errors]` command recursively clones, downloads and updates modules to satisfy the dependencies declared in the `MODULE` files starting from the top-level module. All dependencies to a module must resolve to the same version. If DBT encounters any conflicting version hashes for the same dependency across `MODULE` files, the sync operation fails. If the `--ignore-errors` flag is used, errors related to mismatcing dependency URLs or versions will be ignored.

If the `--update` flag is used, DBT will ignore all previously resolved dependency hashes.

## Build System

### Setup

In order to use the build system features of DBT the `dbt-rules` module must be available. The easiest way to achieve this is to add the following dependency:
```
dbt dep add dbt-rules --url=https://github.com/daedaleanai/dbt-rules.git --version=origin/master
dbt sync
```

### Defining build targets

DBT uses Go for both build target declarations and build rule definitions. DBT thus brings all the advantages and expressivness of a full, strongly-typed programming language to the build system.

In DBT, each *build rule* is a Go `struct` that implements a certain interface. These build rules describe the _general_ steps that need to be executed to produce some kind of an artefact. Examples of build rules provided by DBT are `cc.Library` and `cc.Binary`, which describe steps to build C++ libraries and binaries, respectively.

A *build target* is a concrete instance of such a build rule struct and defines _specific_ steps to produce a _specific_ artefact (e.g. `mylib.a`).
All build targets must be stored in global variables inside `BUILD.go` files to be discovered by DBT.
These `BUILD.go` files can be located anywhere inside a module's directory.
`BUILD.go` files must only contain `import` statements and `var` declarations. Function definitions are not allowed.

Many build rules expect values of type `core.Path` or `core.OutPath` as parameters. Such paths can be obtained by running the `in(name string)`, `ins(names ...string)` and `out(name string)` functions. These functions are auto-generated bt DBT.

The `in` and `ins` functions produce (one or more) `core.Path`s relative to the directory that contains the current `BUILD.go` file. They can be used to reference source files. The `out` function produces `core.OutPath`s in the build directory with the same path relative to the workspace root. There are usually used to refer to build outputs.

Build targets can reference other build targets within the same `BUILD.go` file, across `BUILD.go` files and even across different modules. The usual Go import and visibility rules apply.

The following example shows a simple `BUILD.go` file for a single C++ library and binary:
```
package example

import "dbt-rules/RULES/cc"

var lib = cc.Library{
	Out: out("libexample.a"),
	Srcs: ins(
		"example.cc",
	),
    Hdrs: ins(
		"example.hh",
    ),
}

var example = cc.Binary{
	Out:  out("example"),
	Srcs: ins("main.cc"),
	Deps: []cc.Dep{lib},
}
```

### Building targets

The `dbt build [TARGETS...] [BUILDFLAGS...]` command builds one or multiple targets.
Build targets are identified by the file path to the directory that contains the `BUILD.go` file plus the variable name of the target within the `BUILD.go` file.
Build targets can be specified to the `dbt build` command either relative to the current working directory or relative to the workspace root. In the latter case the target must be prefixed with `//`.

For an example, assume the `BUILD.go` file above was located at `moduleA/path/to/example/BUILD.go` in a workspace. The `example` binary could then be built by running:
* `dbt build //moduleA/path/to/example/example` from any directory inside the workspace.
* `dbt build path/to/example/example` from the `moduleA/` directory.
* `dbt build example/example` from the `moduleA/path/to/` directory.

Multiple build targets can be referenced by using regular expressions. For example, `dbt build //moduleA/path/to/.*` will build all targets defined in the `moduleA/path/to/` directory.

Build flags can be specified using `name=value` syntax. For details see the [relevant section](#build-configuration)].

Running `dbt build` without specifying any targets to build will show a list of all available build targets, as well as all build flags and their current values.

The `dbt clean` command will delete the `BUILD/` directory, which contains all build outputs and intermediate files.

Under the hood, DBT creates a `build.ninja` file to steer the build process. In addition, a `build.sh` file is generated. While this file is not used by DBT itself it contains all commands to build all targets in the workspace and can be used to trigger a full rebuild of all targets when Ninja is not available.

The `dbt build` command supports the following three flags to output additional information about the compilation process:
* `--commands` produces a file that list all commands executed to produce the targets
* `--graph` produces a GraphWiz file with the dependency graph of all produced targets
* `--compdb` produces a [JSON compilation database](https://clang.llvm.org/docs/JSONCompilationDatabase.html) for all targets
The path of the file containing the output is printed by `dbt build` when the respective flag is activated.

### Running targets

The `dbt run [TARGETS...] [BUILDFLAGS...] : [RUNARGS...]` build and runs one or multiple targets.
The targets that are to be built and run must be specified as for the `dbt build` command.

In order for a target to be runnable, its build rule must implement the `Run` interface:
```
type Run interface {
	Run(args []string) string
}
```

The command returned from the `Run` method will be executed by DBT when `dbt run` is called on a target.

Additional arguments can be passed from the command-line to the `Run` method. These arguments must be separated from the targets and build flags with a colon.

### Testing targets

The `dbt test [TARGETS...] [BUILDFLAGS...] : [TESTARGS...]` build and tests one or multiple targets.
The targets that are to be built and tested must be specified as for the `dbt build` command.

In order for a target to be testable, its build rule must implement the `Test` interface:
```
type Test interface {
	Test(args []string) string
}
```

The command returned from the `Test` method will be executed by DBT when `dbt test` is called on a target.

Additional arguments can be passed from the command-line to the `Test` method. These arguments must be separated from the targets and build flags with a colon.

### Creating custom build rules

The `dbt-rules` module provides some basic build rules. However, it is easy to extend DBT with custom rules.

Rules must be implemented in `.go` files that are inside a `RULES/` directory in a module's root directory to be discovered by DBT.

Any Go struct type that implements the `BuildRule` interface qualifies as a build rule.
```
// The BuildRule interface must be implemented by all build rules.
type BuildRule interface {
    Build(ctx Context)
}
```
DBT will call the `Build` method of the struct for each target with a `Context`. 
```
type Context interface {
	AddBuildStep(BuildStep)
	Cwd() OutPath
}
```

The build rule should then use the `AddBuildStep` method of the `Context` to add the necessary build steps for the target.

```
// A build step.
type BuildStep struct {
	Out     OutPath
	Outs    []OutPath
	In      Path
	Ins     []Path
	Depfile OutPath
	Cmd     string
	Script  string
	Descr   string
}
```

Conceptually, each `BuildStep` creates output files (`Outs`) from a set of input files (`Ins`) by executing one or multipe commands. DBT then manages dependencies between input and output files and ensures commands are excuted in the correct order and only executed when necessary (e.g., when inputs to a command change).

The `Out` and `In` fields can be used for convenience when only a single input or output is required.

For each `BuildStep`, either `Script` or `Cmd` can be specified. If `Cmd` is given, DBT will directly execute the command; the content of `Script` will be written to a temporary file first and then executed. This is useful for build steps that consist of many or very long commands.

`Descr` defines the build step description printed by Ninja during the build. `Depfile` is only used to manage dependency information emitted by gcc-like compilers (see the [Ninja documentation](https://ninja-build.org/manual.html#_depfile) for details).

The `Cwd()` function of the `Context` returns an `OutPath` to the `BUILD/OUTPUT-XXXXXX` subdirectory for the current target. This directory may be used by build rules for temporary files.

### Customizing build rule behavior

While all build rule types _must_ implement the `BuildRule` interface, build rules _may_ implement additional interfaces to customize the build behavior.
Currently, the following additional interfaces may be implement bt build rules:

#### Custom build target outputs

By default, DBT will consider all build outputs (i.e., files that are part of `Outs` or `Out` of any `BuildStep`) that are not used as inputs to another build step as the final outputs of a build rule.
The path to these outputs are printed after the build.

Implementing the `BuildOutputs` allows explicitly specifying the printed outputs of a build rule.

```
type BuildOutputs interface {
	Outputs() []Path
}
```

#### Custom build target descriptions

Implementing the `TargetDescription` interface allows generating description for build targets that will be used for shell completion and when listing all build targets (i.e., running `dbt build` without specifying any target to build).

```
type TargetDescription interface {
	Description() string
}
```

### Build configurations

DBT supports different build configuration via build flags. This is useful when building artefects with different build settings.

All flags must be registered into a global variable before they can be used in any build rule.

The following example shows a flag of type `string` being registered:
```
var arch = core.StringFlag{
  // Required: Name of the flag on the command-line
  Name: "arch",
  // Optional: Description of the flag is output by "dbt build"
  Description: "the target architecture",
  // Optional: Provides a default value.
  DefaultFn: func () string {
      return "x86"
  },
  // Optional: DBT will reject any flag value that is not specified here.
  AllowedValues: []string{"x86", "arm", "mips"},
}.Register()
```

The flag's value can now be retrieved by calling `arch.Value()`.

DBT provides `core.StringFlag`s, `core.BoolFlag`s, `core.IntFlag`s and `core.FloatFlag`s. However, only `core.StringFlag`s and `core.BoolFlag`s can have allowed values.

Flag values can be set via the command-line (see [here](#running-builds) for details).

By default, the flag values are only applied to the current invocation of DBT.
However, maintainers of the top-level MODULE file can specify `persist-flags` boolean value,
which impacts all builds in this top-level module.
In this case, flag values are persisted across DBT invocations.
If a flag has been specified via the command-line once that value will be used until a new value is provided via the command-line,
or `dbt clean` command is invoked.

If a flag is not specified on the command-line and has no persisted previous value, the `DefaultFn` will be called to get a default value. If the `DefaultFn` is also not provided, no value can be determined for the flag. In that case DBT will abort the build, since all flags must have a defined value.

A user can set the `persist-flag` option in `~/.config/dbt/config.yaml`, in which case it
will be applied for all modules that do not specify the `persist-flag` option.

### C/C++ rules and cross-compilation

All the rules in dbt-rules/RULES/cc take a an optional `Toolchain` parameter. If the parameter is not specified, the toolchain is selected based on the `cc-toolchain` flag (which defaults to using the native gcc toolchain, i.e. `gcc`, `ld`, ... for native compilation). If you never do cross-compilation, there is nothing to worry about, apart from making sure that `cc-toolchain` is left as the default `native-gcc`.

You can add new toolchains by implementing the `cc.Toolchain` or `cc.GccToolchain` types. These can be used as a value for the `Toolchain` parameter. Optionally, the `RegisterToolchain()` function makes them available from the `cc-toolchain` flag. Toolchains are not just a collection of compile commands, they can also specify standard dependencies, standard includes, and a standard linker script. To this end, you can expand an existing `cc.GccToolchain` with a custom standard library with `GccToolchain.NewWithStdLib()`.

If you have a `cc.Library` that could be compiled with different toolchains *within the same build* (for example if you have some utilities that you want to share between kernel- and user-space programs), mark it with `MultipleToolchains()`. Then the correct one will be selected, based on the `Toolchain` field of what you're building (or the `cc-toolchain` flag if that is not defined). Here is a full example of these concepts at work:

```
// Define a gcc toolchain by populating the struct
val baseToolchain = cc.GccToolchain{ ... }

// Add different stdlibs to baseToolchain to create new toolchains and register them 
val kernelToolchain = cc.RegisterToolchain(baseToolchain.NewWithStdLib(kernelIncludes, kernelDeps, kernelLinkerScript, "kernel-gcc"))
val userToolchain = cc.RegisterToolchain(baseToolchain.NewWithStdLib(userIncludes, userDeps, userLinkerScript, "user-gcc"))

val kernelLib = cc.Library{
	...,
	Toolchain: kernelToolchain,
}

val commonLib = cc.Library{
	...,
	// Don't define Toolchain
}.MultipleToolchains()

val UserBinary = cc.Binary{
	Deps: []cc.Deps{commonLib},
	Toolchain: userToolchain,
}

val KernelBinary = cc.Binary{
	Deps: []cc.Deps{kernelLib, commonLib},
	Toolchain: kernelToolchain,
}
```
