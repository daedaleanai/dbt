package core

import "fmt"

type CopyFile struct {
	From File
	To   OutFile
}

func (copy CopyFile) BuildSteps() []BuildStep {
	return []BuildStep{{
		Out:   copy.To,
		In:    copy.From,
		Cmd:   fmt.Sprintf("cp %s %s", copy.From, copy.To),
		Descr: fmt.Sprintf("CP %s", copy.To.RelPath()),
		Alias: copy.To.RelPath(),
	}}
}
