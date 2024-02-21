package core

import (
	"fmt"
	"path"
	"reflect"
	"strings"
)

// Path represents an on-disk path that is either an input to or an output from a BuildStep (or both).
type Path interface {
	Absolute() string
	Relative() string
	String() string
	WithExt(ext string) OutPath
	WithPrefix(prefix string) OutPath
	WithSuffix(suffix string) OutPath
}

type PkgPath interface {
	SrcDir() string
}

// inPath is a path relative to the workspace source directory.
type inPath struct {
	rel string
	abs string
}

// Absolute returns the absolute path.
func (p inPath) Absolute() string {
	if p.abs != "" {
		return p.abs
	}
	return path.Join(input.SourceDir, p.rel)
}

// Relative returns the path relative to the workspace source directory.
func (p inPath) Relative() string {
	return p.rel
}

// WithExt creates an OutPath with the same relative path and the given extension.
func (p inPath) WithExt(ext string) OutPath {
	return outPath{p.rel}.WithExt(ext)
}

// WithPrefix creates an OutPath with the same relative path and the given prefix.
func (p inPath) WithPrefix(prefix string) OutPath {
	return outPath{p.rel}.WithPrefix(prefix)
}

// WithSuffix creates an OutPath with the same relative path and the given suffix.
func (p inPath) WithSuffix(suffix string) OutPath {
	return outPath{p.rel}.WithSuffix(suffix)
}

// String representation of an inPath is its quoted absolute path.
func (p inPath) String() string {
	return p.Absolute()
}

// OutPath is a path relative to the workspace build directory.
type OutPath interface {
	Path
	forceOutPath()
}

type outPath struct {
	rel string
}

// Absolute returns the absolute path.
func (p outPath) Absolute() string {
	return path.Join(input.OutputDir, p.rel)
}

// Relative returns the path relative to the workspace build directory.
func (p outPath) Relative() string {
	return p.rel
}

// WithExt creates an OutPath with the same relative path and the given extension.
func (p outPath) WithExt(ext string) OutPath {
	oldExt := path.Ext(p.rel)
	newRel := fmt.Sprintf("%s.%s", strings.TrimSuffix(p.rel, oldExt), ext)
	return outPath{newRel}
}

// WithPrefix creates an OutPath with the same relative path and the given prefix.
func (p outPath) WithPrefix(prefix string) OutPath {
	return outPath{path.Join(path.Dir(p.rel), prefix+path.Base(p.rel))}
}

// WithSuffix creates an OutPath with the same relative path and the given suffix.
func (p outPath) WithSuffix(suffix string) OutPath {
	return outPath{p.rel + suffix}
}

// String representation of an OutPath is its quoted absolute path.
func (p outPath) String() string {
	return p.Absolute()
}

// forceOutPath makes sure that inPath or Path cannot be used as OutPath.
func (p outPath) forceOutPath() {}

// GlobalPath is a global path.
type GlobalPath interface {
	Absolute() string
}

type globalPath struct {
	abs string
}

// Absolute returns absolute path.
func (p globalPath) Absolute() string {
	return p.abs
}

// String representation of a globalPath is its quoted absolute path.
func (p globalPath) String() string {
	return p.Absolute()
}

// NewInPath creates an inPath for a path relativ to the source directory.
func NewInPath(pkg PkgPath, p string) Path {
	return inPath{path.Join(reflect.TypeOf(pkg).PkgPath(), p), path.Join(pkg.SrcDir(), p)}
}

// NewOutPath creates an OutPath for a path relativ to the build directory.
func NewOutPath(pkg interface{}, p string) OutPath {
	return outPath{path.Join(reflect.TypeOf(pkg).PkgPath(), p)}
}

// NewGlobalPath creates a globalPath.
func NewGlobalPath(p string) GlobalPath {
	return globalPath{p}
}

// BuildPath returns a path relative to the build directory.
func BuildPath(p string) OutPath {
	return outPath{p}
}

// SourcePath returns a path relative to the source directory.
func SourcePath(p string) Path {
	return inPath{p, ""}
}
