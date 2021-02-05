package core

import (
	"fmt"
	"strings"
)

// BuildStep represents one build step (i.e., one build command).
// Each BuildStep produces `Out` from `Ins` and `In` by running `Cmd`.
type BuildStep struct {
	Out     OutPath
	In      Path
	Ins     Paths
	Depfile *OutPath
	Cmd     string
	Descr   string
	Alias   string
}

var nextRuleID = 1

func ninjaEscape(s string) string {
	return strings.ReplaceAll(s, " ", "$ ")
}

// Print outputs a Ninja build rule for the BuildStep.
func (step BuildStep) Print() {
	ins := []string{}
	for _, in := range step.Ins {
		ins = append(ins, ninjaEscape(in.Absolute()))
	}
	if step.In != nil {
		ins = append(ins, ninjaEscape(step.In.Absolute()))
	}

	alias := ninjaEscape(step.Alias)
	out := ninjaEscape(step.Out.Absolute())

	fmt.Printf("rule r%d\n", nextRuleID)
	if step.Depfile != nil {
		depfile := ninjaEscape(step.Depfile.Absolute())
		fmt.Printf("  depfile = %s\n", depfile)
	}
	fmt.Printf("  command = %s\n", step.Cmd)
	if step.Descr != "" {
		fmt.Printf("  description = %s\n", step.Descr)
	}
	fmt.Print("\n")
	fmt.Printf("build %s: r%d %s\n", out, nextRuleID, strings.Join(ins, " "))
	if alias != "" {
		fmt.Print("\n")
		fmt.Printf("build %s: phony %s\n", alias, out)
	}
	fmt.Print("\n\n")

	nextRuleID++
}
