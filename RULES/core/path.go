package core

import (
	"fmt"
	"path"
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

// Paths represents a list of Paths.
type Paths []Path

func (ps Paths) String() string {
	paths := []string{}
	for _, p := range ps {
		paths = append(paths, fmt.Sprint(p))
	}
	return strings.Join(paths, " ")
}

// inPath is a path relative to the workspace source directory.
type inPath struct {
	rel string
}

// Absolute returns the absolute path.
func (p inPath) Absolute() string {
	return path.Join(SourceDir(), p.rel)
}

// Relative returns the path relative to the workspace source directory.
func (p inPath) Relative() string {
	return p.rel
}

// WithExt creates an OutPath with the same relative path and the given extension.
func (p inPath) WithExt(ext string) OutPath {
	return OutPath{p.rel}.WithExt(ext)
}

// WithPrefix creates an OutPath with the same relative path and the given prefix.
func (p inPath) WithPrefix(prefix string) OutPath {
	return OutPath{p.rel}.WithPrefix(prefix)
}

// WithSuffix creates an OutPath with the same relative path and the given suffix.
func (p inPath) WithSuffix(suffix string) OutPath {
	return OutPath{p.rel}.WithSuffix(suffix)
}

// String representation of an inPath is its quoted absolute path.
func (p inPath) String() string {
	return fmt.Sprintf("\"%s\"", p.Absolute())
}

// OutPath is a path relative to the workspace build directory.
type OutPath struct {
	rel string
}

// Absolute returns the absolute path.
func (p OutPath) Absolute() string {
	return path.Join(BuildDir(), p.rel)
}

// Relative returns the path relative to the workspace build directory.
func (p OutPath) Relative() string {
	return p.rel
}

// WithExt creates an OutPath with the same relative path and the given extension.
func (p OutPath) WithExt(ext string) OutPath {
	oldExt := path.Ext(p.rel)
	newRel := fmt.Sprintf("%s.%s", strings.TrimSuffix(p.rel, oldExt), ext)
	return OutPath{newRel}
}

// WithPrefix creates an OutPath with the same relative path and the given prefix.
func (p OutPath) WithPrefix(prefix string) OutPath {
	return OutPath{path.Join(path.Dir(p.rel), prefix+path.Base(p.rel))}
}

// WithSuffix creates an OutPath with the same relative path and the given suffix.
func (p OutPath) WithSuffix(suffix string) OutPath {
	return OutPath{p.rel + suffix}
}

// String representation of an OutPath is its quoted absolute path.
func (p OutPath) String() string {
	return fmt.Sprintf("\"%s\"", p.Absolute())
}

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
	return fmt.Sprintf("\"%s\"", p.Absolute())
}

// NewInPath creates an inPath for a path relativ to the source directory.
func NewInPath(p string) Path {
	return inPath{p}
}

// NewOutPath creates an OutPath for a path relativ to the build directory.
func NewOutPath(p string) OutPath {
	return OutPath{p}
}

// NewGlobalPath creates a globalPath.
func NewGlobalPath(p string) GlobalPath {
	return globalPath{p}
}
