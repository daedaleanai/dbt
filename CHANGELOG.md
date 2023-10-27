# Changelog

### v1.4.0

- Switched to semantic versioning of the binaries.
- The MODULE file format is no longer directly linked to the minor version of DBT,
  versions 1 through 3 are supported.
- The "persist-flags" configuration can be enforced for a project in top level MODULE file.
  The field is retrofitted into all known MODULE file formats.
  The value specified in the field is applied to all builds of the project.
