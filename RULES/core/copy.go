package core

import "fmt"

// CopyFile copies a single file.
type CopyFile struct {
	From Path
	To   OutPath
}

// Build for CopyFile.
func (copy CopyFile) Build(ctx Context) OutPath {
	ctx.AddBuildStep(BuildStep{
		Out:   copy.To,
		In:    copy.From,
		Cmd:   fmt.Sprintf("cp %s %s", copy.From, copy.To),
		Descr: fmt.Sprintf("CP %s", copy.To.Relative()),
	})
	return copy.To
}
