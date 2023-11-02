package assets

import (
	"embed"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed statics/*
var Statics embed.FS

var Templates = template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))

type InitFileTemplate struct {
	Package   string
	Vars      []string
	SourceDir string
}

type MainFileTemplate struct {
	RequiredGoVersionMajor uint64
	RequiredGoVersionMinor uint64
	Packages               []string
}

type GoModTemplate struct {
	RequiredGoVersionMajor uint64
	RequiredGoVersionMinor uint64
	Module                 string
	Prefix                 string
	Deps                   []string
}
