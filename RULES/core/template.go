package core

import (
	"fmt"
	"strings"
)

type ExpandTemplate struct {
	Out           OutFile
	Template      File
	Substitutions map[string]string
}

func (tmpl ExpandTemplate) BuildSteps() []BuildStep {
	substitutions := []string{}
	for old, new := range tmpl.Substitutions {
		substitutions = append(substitutions, fmt.Sprintf("-e 's/%s/%s/g'", old, new))
	}
	cmd := fmt.Sprintf("sed %s %s > %s", strings.Join(substitutions, " "), tmpl.Template, tmpl.Out)
	return []BuildStep{{
		Out:   tmpl.Out,
		In:    tmpl.Template,
		Cmd:   cmd,
		Descr: fmt.Sprintf("TEMPLATE %s", tmpl.Out.RelPath()),
		Alias: tmpl.Out.RelPath(),
	}}
}
