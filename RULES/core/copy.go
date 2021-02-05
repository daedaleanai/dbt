package core

import "fmt"

// CopyFile copies a single file.
type CopyFile struct {
	From Path
	To   OutPath
}

// BuildSteps for CopyFile.
func (copy CopyFile) BuildSteps() []BuildStep {
	return []BuildStep{{
		Out:   copy.To,
		In:    copy.From,
		Cmd:   fmt.Sprintf("cp %s %s", copy.From, copy.To),
		Descr: fmt.Sprintf("CP %s", copy.To.Relative()),
		Alias: copy.To.Relative(),
	}}
}
