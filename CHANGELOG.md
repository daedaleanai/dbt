# Changelog

### v2.0.0

- Switched to semantic versioning of the binaries.
- The default value of persist-flags has been changed from `true` to `false`.
  This value can be overriden in MODULE file of a project or in the user's configuration.
  The former takes precedence.
- The MODULE file format is no longer directly linked to the minor version of DBT,
  versions 1 through 3 are supported.
- Updated the minimum required Go dependency to 1.18.
- `go build --tags=semver-override=vX.Y.Z-dev` should be used when building DBT from
  source on an untagged branch and/or with local changes.

### v1.3.19

- The "persist-flags" configuration can be enforced for a project in top level MODULE file.
  The field is retrofitted into all known MODULE file formats.
  The value specified in the field is applied to all builds of the project.
