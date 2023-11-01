package cmd

import (
	"embed"
	"text/template"
)

//go:embed assets/*
var assets embed.FS

var templates = template.Must(template.ParseFS(assets, "assets/*.tmpl"))

type initFileParams struct {
	Package   string
	Vars      []string
	SourceDir string
}

type mainFileParams struct {
	RequiredGoVersionMajor uint64
	RequiredGoVersionMinor uint64
	Packages               []string
}

type goModParams struct {
	RequiredGoVersionMajor uint64
	RequiredGoVersionMinor uint64
	Module                 string
	Prefix                 string
	Deps                   []string
}
