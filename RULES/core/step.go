package core

import (
	"fmt"
	"strings"
)

// BuildStep represents one build step (i.e., one build command).
// Each BuildStep produces "Out" from "Ins" and "In" by running "Cmd".
type BuildStep struct {
	Out   OutFile
	In    File
	Ins   Files
	Cmd   string
	Descr string
	Alias string
}

var nextRuleId = 1

func ninjaEscape(s string) string {
	return strings.ReplaceAll(s, " ", "$ ")
}

// Print outputs a Ninja build rule for the BuildStep.
func (step BuildStep) Print() {
	ins := []string{}
	for _, in := range step.Ins {
		ins = append(ins, ninjaEscape(in.Path()))
	}
	if step.In != nil {
		ins = append(ins, ninjaEscape(step.In.Path()))
	}

	alias := ninjaEscape(step.Alias)
	out := ninjaEscape(step.Out.Path())

	fmt.Printf("rule r%d\n", nextRuleId)
	fmt.Printf("  command = %s\n", step.Cmd)
	if step.Descr != "" {
		fmt.Printf("  description = %s\n", step.Descr)
	}
	fmt.Print("\n")
	fmt.Printf("build %s: r%d %s\n", out, nextRuleId, strings.Join(ins, " "))
	if alias != "" {
		fmt.Print("\n")
		fmt.Printf("build %s: phony %s\n", alias, out)
	}
	fmt.Print("\n\n")

	nextRuleId++
}
