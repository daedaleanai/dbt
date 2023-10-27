# Changelog

### v2.0.0-pre1

- Updated the minimum required Go dependency to 1.18.
- `dbt --version` output format has changed, and can now contain semver pre-release and build components.
- The default value of persist-flags has been changed from `true` to `false`.
  This value can be overriden in MODULE file of a project or in the user's configuration.
  The former takes precedence.
- The build version is deduced from Git tags. It is a hard failure if the deduction fails.  
  Use `go build --tags=semver-override=vX.Y.Z-dev` when building DBT from source on
  an untagged branch and/or with local changes.

### v1.4.0

- Switched to semantic versioning of the binaries.
- The MODULE file format is no longer directly linked to the minor version of DBT,
  versions 1 through 3 are supported.
- The "persist-flags" configuration can be enforced for a project in top level MODULE file.
  The field is retrofitted into all known MODULE file formats.
  The value specified in the field is applied to all builds of the project,
  overriding the user configuration.
