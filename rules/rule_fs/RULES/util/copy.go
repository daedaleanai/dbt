package util

import (
	"fmt"

	"dbt-rules/RULES/core"
)

func init() {
	core.AssertIsBuildableTarget(&CopyFile{})
}

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
