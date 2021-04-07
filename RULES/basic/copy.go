package basic

import (
	"fmt"

	"dbt/RULES/core"
)

// CopyFile copies a single file.
type CopyFile struct {
	From core.Path
	To   core.OutPath
}

// Build for CopyFile.
func (copy CopyFile) Build(ctx core.Context) {
	ctx.AddBuildStep(core.BuildStep{
		Out:   copy.To,
		In:    copy.From,
		Cmd:   fmt.Sprintf("cp %q %q", copy.From, copy.To),
		Descr: fmt.Sprintf("CP %s", copy.To.Relative()),
	})
}

func (copy CopyFile) Output() core.OutPath {
	return copy.To
}
