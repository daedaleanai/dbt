# Changelog

### Future v3.2.0

- Minor fix to the handling of Go modules listed in the `cpp` layout, which now does not include the repository
basename as a Go module.
- Add a `dbt lsp` command that implements the driver protocol of the [go/packages](golang.org/x/tools/go/packages) 
repository. This functionality may be used to perform correct package resolution for dbt files. An 
integration example in neovim can be found [here](https://github.com/Javier-varez/dbt-nvim/blob/ed9b73547b2f056e8043e5817a6451f9a01e74f1/lua/dbt-nvim/lsp/init.lua#L10).

### v3.1.0 (also: v3.1.0-rc1)

- Add `-k` argument to dbt to specify the number of failures after which ninja must stop execution.
If this value is 0, ninja only stops when it has built all targets or the remaining targets depend
on other failed targets. Defaults to 1 for backwards compatibility.

### v3.0.0 (also: v3.0.0-rc1)

- Target filtering moved to dbt-rules. Minimum dbt-rules version is 2.0.0
- Coverage and analyze commands removed. They could be implemented in project-specific RULES

### v2.0.3

- Bug fix: The name of the package must have been changed earlier, it is done now.
- Tags and versions v2.0.1 and v2.0.2 had been used for much older unrelated releases,
  they are cursed and should not be used.

### v2.0.0 (also: v2.0.0-rc2)

- Refactoring: Ordered data structures introduced and used throughout the implementation of `dbt build`.

### v2.0.0-rc1

- Updated the minimum required Go dependency to 1.18 for both DBT build and DBT rules.
- `dbt --version` output format has changed, and can now contain semver pre-release and build components.
- The default value of persist-flags has been changed from `true` to `false`.
  This value can be overriden in MODULE file of a project or in the user's configuration.
  The former takes precedence.
- The build version is deduced from Git tags. It is a hard failure if the deduction fails.
  Use `go build --tags=semver-override=vX.Y.Z-dev` when building DBT from source on
  an untagged branch and/or with local changes.
- `dbt sync --strict` hardened: it no longer overwrites the top level MODULE file.
  While normally it should have been the case already, bugs could inadvertently break this assumption,
  see e.g. v1.4.1 release.
- Remove restriction to run ninja with 1 thread for the coverage command. Instead, give the user the
  option to set the number of threads and default to as many threads as cores (-1).

### v1.4.1

- Bug fix (introduced in v1.4.0): The "persist-flags" configuration flag erroneously appeared in the
  MODULE file as a side effect of `dbt sync`.

### v1.4.0

- Switched to semantic versioning of the binaries.
- The MODULE file format is no longer directly linked to the minor version of DBT,
  versions 1 through 3 are supported.
- The "persist-flags" configuration can be enforced for a project in top level MODULE file.
  The field is retrofitted into all known MODULE file formats.
  The value specified in the field is applied to all builds of the project,
  overriding the user configuration.
