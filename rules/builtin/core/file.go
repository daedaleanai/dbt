package core

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
)

func sourceDir() string {
	return os.Args[1]
}

func buildDir() string {
	return os.Args[2]
}

// File represents an on-disk file that is either an input to or an output from a BuildStep (or both).
type File interface {
	Empty() bool
	Path() string
	RelPath() string
	WithExt(ext string) OutFile
	WithSuffix(suffix string) OutFile
}

// Files represents a group of Files.
type Files []File

func (fs Files) String() string {
	paths := []string{}
	for _, f := range fs {
		paths = append(paths, fmt.Sprint(f))
	}
	return strings.Join(paths, " ")
}

type files interface {
	Files() Files
}

// Files implements the files interface for a group of files.
func (fs Files) Files() Files {
	return fs
}

// Flatten flattens a list of individual files or groups of files.
func Flatten(fss ...files) Files {
	files := Files{}
	for _, fs := range fss {
		files = append(files, fs.Files()...)
	}
	return files
}

// InFile represents a file relative to the source directory.
type InFile struct {
	relPath string
}

// Empty returns whether the file path is empty.
func (f InFile) Empty() bool {
	return f.relPath == ""
}

// Path returns the file's absolute path.
func (f InFile) Path() string {
	return path.Join(sourceDir(), f.relPath)
}

// RelPath returns the file's path relative to the source directory.
func (f InFile) RelPath() string {
	return f.relPath
}

// WithExt creates an OutFile with the same relative path and the given file extension.
func (f InFile) WithExt(ext string) OutFile {
	return OutFile{f.relPath}.WithExt(ext)
}

// WithSuffix creates an OutFile with the same relative path and the given suffix.
func (f InFile) WithSuffix(suffix string) OutFile {
	return OutFile{f.relPath}.WithSuffix(suffix)
}

func (f InFile) String() string {
	return fmt.Sprintf("\"%s\"", f.Path())
}

// InFile represents a file relative to the build directory.
type OutFile struct {
	relPath string
}

// Empty returns whether the file path is empty.
func (f OutFile) Empty() bool {
	return f.relPath == ""
}

// Path returns the file's absolute path.
func (f OutFile) Path() string {
	return path.Join(buildDir(), f.relPath)
}

// RelPath returns the file's path relative to the build directory.
func (f OutFile) RelPath() string {
	return f.relPath
}

// WithExt creates an OutFile with the same relative path and the given file extension.
func (f OutFile) WithExt(ext string) OutFile {
	oldExt := path.Ext(f.relPath)
	relPath := fmt.Sprintf("%s.%s", strings.TrimSuffix(f.relPath, oldExt), ext)
	return OutFile{relPath}
}

// WithSuffix creates an OutFile with the same relative path and the given suffix.
func (f OutFile) WithSuffix(suffix string) OutFile {
	return OutFile{f.relPath + suffix}
}

func (f OutFile) String() string {
	return fmt.Sprintf("\"%s\"", f.Path())
}

// NewInFile creates an InFile for a file relativ to the package directory of "pkg".
func NewInFile(name string, pkg interface{}) InFile {
	pkgPath := reflect.TypeOf(pkg).PkgPath()
	p := path.Join(strings.TrimPrefix(pkgPath, "_/"), name)
	return InFile{p}
}

// NewOutFile creates an OutFile for a file relativ to the package directory of "pkg".
func NewOutFile(name string, pkg interface{}) OutFile {
	pkgPath := reflect.TypeOf(pkg).PkgPath()
	p := path.Join(strings.TrimPrefix(pkgPath, "_/"), name)
	return OutFile{p}
}
