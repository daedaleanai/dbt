package util

import (
	"fmt"
	"sort"
	"strings"

	"dbt-rules/RULES/core"
)

func init() {
	core.AssertIsBuildableTarget(&ExpandTemplate{})
}

// ExpandTemplate expands `Template` by performing `Substitutions` and storing the result in `Out`.
type ExpandTemplate struct {
	Out           core.OutPath
	Template      core.Path
	Substitutions map[string]string
}

// BuildSteps for ExpandTemplate.
func (tmpl ExpandTemplate) Build(ctx core.Context) {
	substitutions := []string{}
	for old, new := range tmpl.Substitutions {
		substitutions = append(substitutions, fmt.Sprintf("-e 's/%s/%s/g'", old, new))
	}
	sort.Strings(substitutions)
	cmd := fmt.Sprintf("sed %s %q > %q", strings.Join(substitutions, " "), tmpl.Template, tmpl.Out)
	ctx.AddBuildStep(core.BuildStep{
		Out:   tmpl.Out,
		In:    tmpl.Template,
		Cmd:   cmd,
		Descr: fmt.Sprintf("TEMPLATE %s", tmpl.Out.Relative()),
	})
}
