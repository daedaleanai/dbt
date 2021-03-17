package core

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Context interface {
	Initialize()
	AddTarget(name string, target interface{})
	AddBuildStep(BuildStep)
}

type NinjaContext struct {
	nextRuleID int
}

func ninjaEscape(s string) string {
	return strings.ReplaceAll(s, " ", "$ ")
}

type buildsOne interface {
	Build(ctx Context) OutPath
}

type buildsMultiple interface {
	Build(ctx Context) OutPaths
}

func (ctx *NinjaContext) Initialize() {
	fmt.Printf("build __phony__: phony\n\n")
}

func (ctx *NinjaContext) AddTarget(name string, target interface{}) {
	currentTarget = name
	outs := OutPaths{}

	if iface, ok := target.(buildsOne); ok {
		outs = OutPaths{iface.Build(ctx)}
	}

	if iface, ok := target.(buildsMultiple); ok {
		outs = iface.Build(ctx)
	}

	if len(outs) == 0 {
		return
	}

	relPaths := []string{}
	ninjaPaths := []string{}
	for _, out := range outs {
		relPath, _ := filepath.Rel(WorkingDir(), out.Absolute())
		relPaths = append(relPaths, relPath)
		ninjaPaths = append(ninjaPaths, ninjaEscape(out.Absolute()))
	}

	fmt.Printf("rule r%d\n", ctx.nextRuleID)
	fmt.Printf("  command = echo \"%s\"\n", strings.Join(relPaths, "\\n"))
	fmt.Printf("  description = Created %s:", name)
	fmt.Printf("\n")
	fmt.Printf("build %s: r%d %s __phony__\n", name, ctx.nextRuleID, strings.Join(ninjaPaths, " "))
	fmt.Printf("\n")
	fmt.Printf("\n")

	ctx.nextRuleID++
}

func (ctx *NinjaContext) AddBuildStep(step BuildStep) {
	ins := []string{}
	for _, in := range step.Ins {
		ins = append(ins, ninjaEscape(in.Absolute()))
	}
	if step.In != nil {
		ins = append(ins, ninjaEscape(step.In.Absolute()))
	}

	outs := []string{}
	for _, out := range step.Outs {
		outs = append(outs, ninjaEscape(out.Absolute()))
	}
	if step.Out != nil {
		outs = append(outs, ninjaEscape(step.Out.Absolute()))
	}

	fmt.Printf("rule r%d\n", ctx.nextRuleID)
	if step.Depfile != nil {
		depfile := ninjaEscape(step.Depfile.Absolute())
		fmt.Printf("  depfile = %s\n", depfile)
	}
	fmt.Printf("  command = %s\n", step.Cmd)
	if step.Descr != "" {
		fmt.Printf("  description = %s\n", step.Descr)
	}
	fmt.Print("\n")
	fmt.Printf("build %s: r%d %s\n", strings.Join(outs, " "), ctx.nextRuleID, strings.Join(ins, " "))
	fmt.Print("\n\n")

	ctx.nextRuleID++
}

type ListTargetsContext struct{}

func (ctx *ListTargetsContext) Initialize() {}

func (ctx *ListTargetsContext) AddTarget(name string, target interface{}) {
	_, okOne := target.(buildsOne)
	_, okMultiple := target.(buildsMultiple)
	if okOne || okMultiple {
		fmt.Println(name)
	}
}

func (ctx *ListTargetsContext) AddBuildStep(step BuildStep) {}
