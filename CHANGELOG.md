# Changelog

### Unreleased

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
