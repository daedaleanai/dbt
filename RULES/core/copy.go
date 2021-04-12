package core

import "fmt"

// CopyFile copies a single file.
type CopyFile struct {
	From Path
	To   OutPath
}

// Build for CopyFile.
func (copy CopyFile) Build(ctx Context) {
	ctx.AddBuildStep(BuildStep{
		Out:   copy.To,
		In:    copy.From,
		Cmd:   fmt.Sprintf("cp %q %q", copy.From, copy.To),
		Descr: fmt.Sprintf("CP %s", copy.To.Relative()),
	})
}

func (copy CopyFile) Output() OutPath {
	return copy.To
}
