package core

import (
	"fmt"
	"path"
	"strings"
)

// File represents an on-disk file that is either an input to or an output from a BuildStep (or both).
type File interface {
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

// inFile represents a file relative to the workspace source directory.
type inFile struct {
	relPath string
}

// Path returns the file's absolute path.
func (f inFile) Path() string {
	return path.Join(SourceDir(), f.relPath)
}

// RelPath returns the file's path relative to the source directory.
func (f inFile) RelPath() string {
	return f.relPath
}

// WithExt creates an OutFile with the same relative path and the given file extension.
func (f inFile) WithExt(ext string) OutFile {
	return OutFile{f.relPath}.WithExt(ext)
}

// WithSuffix creates an OutFile with the same relative path and the given suffix.
func (f inFile) WithSuffix(suffix string) OutFile {
	return OutFile{f.relPath}.WithSuffix(suffix)
}

func (f inFile) String() string {
	return fmt.Sprintf("\"%s\"", f.Path())
}

// OutFile represents a file relative to the workspace build directory.
type OutFile struct {
	relPath string
}

// Path returns the file's absolute path.
func (f OutFile) Path() string {
	return path.Join(BuildDir(), f.relPath)
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

// GlobalFile represents a global file.
type GlobalFile interface {
	Path() string
}

type globalFile struct {
	absPath string
}

// Path returns the file's absolute path.
func (f globalFile) Path() string {
	return f.absPath
}

// NewInFile creates an inFile for a file relativ to the source directory.
func NewInFile(p string) File {
	return inFile{p}
}

// NewOutFile creates an OutFile for a file relativ to the build directory.
func NewOutFile(p string) OutFile {
	return OutFile{p}
}

// NewGlobalFile creates a globalFile.
func NewGlobalFile(p string) GlobalFile {
	return globalFile{p}
}
