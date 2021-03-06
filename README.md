# Daedalean Build Tool

The Daedalean Build Tool (DBT) provides dependency management and build system features in a single tool. The dependency management allows setting up workspaces consisting of multiple modules and managing dependencies between different modules. The build system components allows defining build targets and custom build rules in Go.

## Requirements

DBT requires the following tools to be installed on your system:

* go (>= v1.13)
* git
* ninja

## Installation

You can install the latest version of DBT by running:
```
go get github.com/daedaleanai/dbt
```

## General remarks

* All DBT commands have a `-v` / `--verbose` flag to enable debug output.
* The `dbt version` command prints the current version of the tool.
* DBT supports shell completion for `bash`, `zsh`, and `fish` shells. Run `dbt completion bash|zsh|fish` to get the respective completion script.
* The auto-generated Go documentation for this repository can be found [here](https://pkg.go.dev/github.com/daedaleanai/dbt).
* The auto-generated Go documentation for the `dbt-rules` repository can be found [here](https://pkg.go.dev/github.com/daedaleanai/dbt-rules).

## Dependency management

In DBT, dependency management is centered around the concept of modules. DBT currently supports two types of modules: Git repositories and `.tar.gz` archives.

Each module contains a `MODULE` file in its root directory to declare its dependencies on other modules. Modules always depend on a _named version_ of another module. In case of a Git dependency, this can be a branch name, tag or commit hash. `.tar.gz` archive dependencies only have a single version called `master`.

When a dependency is resolved for the first time (e.g., when running the `dbt sync` command), the dependency version is resolved to a hash that uniquely identifies a snapshot of the dependency. For Git dependencies this is the commit hash, for `.tar.gz` archives this is the `sha256` hash of the archives content.

The resolved hash is then added to the `MODULE` file of the dependent module. To guarantee reproducible builds, DBT will always use the hash from the `MODULE` file to resolve a dependency, if it is available. In order to update these hashes (e.g., when a dependency on a Git branch should reflect new commits), the `dbt dep update` [command](#updating-dependency-version-hashes) must be used.

## Directory structure

There is no explicit concept of workspaces. Instead, each module can "become" a workspace when running the `dbt sync` command in the module directory. This module is then called the top-level module or workspace. The `dbt sync` command creates a `DEPS/` directory in the workspace's root directory. All direct and transitive dependencies will be stored inside the `DEPS/` directory. Furthermore, a symlink from the workspace root directory into the `DEPS/` directory is created. The symlink ensures that all modules can access their dependencies as sibling directories regardles of which module acts as the workspace.

### Manipulating MODULE files

`MODULE` files should rarely (if ever) be edited by hand. Instead, the following commands should be used to add, remove and update dependencies.
The following commands always act on the `MODULE` file of the current module (according to the working directory) and not necessarily the top-level module. The commands only edit `MODULE` files but will never download, delete, clone or in any other way change any other than the current module.

#### Adding a dependency

To add a Git repository dependency to the current module run:
```
dbt dep add git [NAME] URL VERSION
```

To add a `.tar.gz` archive dependency to the current module run:
```
dbt dep add tar [NAME] URL
```

The `NAME` parameter determines the name of the module directory inside the `DEPS/` directory. It is derived from the `URL` if omitted.
In order to change the version of the dependency (e.g., to depend on another branch of a dependency), simply rerun the command. 

#### Removing a dependency

To remove a dependency from the current module run:
```
dbt dep remove NAME|URL
```

#### Updating dependency version hashes

Once a dependency version is resolved to a hash (i.e., after running `dbt sync`), that dependency is then fixed to that version hash until it is explicitly updated. That means, dependencies on Git branches will not automatically resolve to the tip of that branch.

The `dbt dep update [--all] [MODULES...]` command can be used to re-run the resolution of _named_ versions to hashes. The command will only update dependency entries to `MODULES`. If no `MODULES` are spedified, all dependency entries are updated.

By default, the command will only execute on the current module. If the `--all` flag is specified, the command will execute on all modules in the workspace.

Examples:
* `dbt dep update` will update all dependency entries of the current module
* `dbt dep update moduleA moduleB` will update the dependency entries for dependencies to modules `moduleA` and `moduleB` for the current module
* `dbt dep update --all` will update all dependency entries of all modules in the workspace
* `dbt dep update --all moduleC` will update the dependency entries for dependencies to module `moduleC` for all modules in the workspace

### Module initialization

If a module has a `SETUP.go` file in its root directory, DBT will run the `SETUP.go` file when the module is initially cloned or downloaded. This mechanism can be be used to initialize modules (e.g. install git hooks).

### The sync command

The `dbt sync [--master]` command recursively clones, downloads and updates modules to satisfy the dependencies declared in the `MODULE` files starting from the top-level module. All dependencies to a module must resolve to the same version hash. If DBT encounters any conflicting version hashes for the same dependency across `MODULE` files, the sync operation fails.

If the `--master` flag is specified, DBT will ignore all versions specified in `MODULE` files and use `master` as the version for all dependencies.

### The fetch command

`dbt fetch` will run `git fetch` in all Git modules in the workspace. Modules that have uncommited changes will be skipped. This is mostly useful in combination with the `git dep update` command.

### The status command

The `dbt status` command prints a summary of all currently available modules and their current versions, and reports any unsatisied dependencies. The command will never perform any changes on any module. 

## Build System

### Setup

In order to use the build system features of DBT the `dbt-rules` module must be available. The easiest way to achive this is to add the following dependency:
```
dbt dep add git https://github.com/daedaleanai/dbt-rules.git origin/master
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

### Running builds

The `dbt build [TARGETS...] [BUILDFLAGS...]` build one or multiple targets.
Build targets are identified by the file path to the directory that contains the `BUILD.go` file plus the variable name of the target within the `BUILD.go` file.
Build targets can be specified to the `sbt build` command either relative to the current working directory or relative to the workspace root. In the latter case the target must be prefixed with `//`.

For an example, assume the `BUILD.go` file above was located at `moduleA/path/to/example/BUILD.go` in a workspace. The `example` binary could then be built by running:
* `dbt build //moduleA/path/to/example/example` from any directory inside the workspace.
* `dbt build path/to/example/example` from the `moduleA/` directory.
* `dbt build example/example` from the `moduleA/path/to/` directory.

Multiple build targets can be referenced by using the `...` suffix. For example, `dbt build //moduleA/path/to/...` will build all target that match the prefix `//moduleA/path/to/` (i.e., all targets defined in the `moduleA/path/to/` directory).

Build flags can be specified using `name=value` syntax. For details see the [relevant section](#build-configuration)].

Running `dbt build` without specifying any targets to build will show a list of all available build targets, as well as all build flags and their current values.

The `dbt clean` command will delete the `BUILD/` directory.

Under the hood, DBT creates a `build.ninja` file to steer the build process. In addition, a `build.sh` file is generated. While this file is not used by DBT itself it contains all commands to build all targets in the workspace and can be used to trigger a full rebuild of all targets when Ninja is not available.

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

DBT supports different build configuration via build flags. This is useful when building artefects for different target architectures.

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

Flag values can be set via the command-line (see [here](#running-builds) for details). Once specified, flag values are persisted across DBT invocations. If a flag has been specified via the command-line once that value will be used until a new value is provided via the command-line.

If a flag is not specified on the command-line and has no persisted previous value, the `DefaultFn` will be called to get a default value. If the `DefaultFn` is also not provided, no value can be determined fot the flag. In that case DBT will abort the build, since all flags must have a defined value.

DBT will create a separate `BUILD/OUTPUT-XXXXXX` directory for each build configuration (i.e., set of build flag values). This allows for intermediate build results to be reused even when switching back-and-forth between different build configurations.
