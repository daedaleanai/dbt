package core

import (
	"fmt"
	"strings"
)

// ExpandTemplate expands `Template` by performing `Substitutions` and storing the result in `Out`.
type ExpandTemplate struct {
	Out           OutPath
	Template      Path
	Substitutions map[string]string
}

// BuildSteps for ExpandTemplate.
func (tmpl ExpandTemplate) Build(ctx Context) OutPath {
	substitutions := []string{}
	for old, new := range tmpl.Substitutions {
		substitutions = append(substitutions, fmt.Sprintf("-e 's/%s/%s/g'", old, new))
	}
	cmd := fmt.Sprintf("sed %s %s > %s", strings.Join(substitutions, " "), tmpl.Template, tmpl.Out)
	ctx.AddBuildStep(BuildStep{
		Out:   tmpl.Out,
		In:    tmpl.Template,
		Cmd:   cmd,
		Descr: fmt.Sprintf("TEMPLATE %s", tmpl.Out.Relative()),
	})
	return tmpl.Out
}
