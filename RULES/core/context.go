package core

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type Context interface {
	AddBuildStep(BuildStep)
	Cwd() Path
}

type NinjaContext struct {
	writer     io.Writer
	nextRuleID int
	cwd        OutPath
}

func NewNinjaContext(writer io.Writer) *NinjaContext {
	return &NinjaContext{writer, 0, outPath{}}
}

func ninjaEscape(s string) string {
	return strings.ReplaceAll(s, " ", "$ ")
}

type buildable interface {
	Build(ctx Context)
}

type output interface {
	Output() OutPath
}

type outputs interface {
	Outputs() OutPaths
}

func (ctx *NinjaContext) Initialize() {
	fmt.Fprintf(ctx.writer, "build __phony__: phony\n\n")
}

func (ctx *NinjaContext) AddTarget(name string, target interface{}, cwd OutPath) {
	currentTarget = name
	ctx.cwd = cwd

	if iface, ok := target.(buildable); ok {
		iface.Build(ctx)
	}

	outs := OutPaths{}
	if iface, ok := target.(outputs); ok {
		outs = iface.Outputs()
	}
	if iface, ok := target.(output); ok {
		outs = append(outs, iface.Output())
	}

	if len(outs) == 0 {
		return
	}

	relPaths := []string{}
	ninjaPaths := []string{}
	for _, out := range outs {
		relPath, _ := filepath.Rel(workingDir(), out.Absolute())
		relPaths = append(relPaths, relPath)
		ninjaPaths = append(ninjaPaths, ninjaEscape(out.Absolute()))
	}

	fmt.Fprintf(ctx.writer, "rule r%d\n", ctx.nextRuleID)
	fmt.Fprintf(ctx.writer, "  command = echo \"%s\"\n", strings.Join(relPaths, "\\n"))
	fmt.Fprintf(ctx.writer, "  description = Created %s:", name)
	fmt.Fprintf(ctx.writer, "\n")
	fmt.Fprintf(ctx.writer, "build %s: r%d %s __phony__\n", name, ctx.nextRuleID, strings.Join(ninjaPaths, " "))
	fmt.Fprintf(ctx.writer, "\n")
	fmt.Fprintf(ctx.writer, "\n")

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

	fmt.Fprintf(ctx.writer, "rule r%d\n", ctx.nextRuleID)
	if step.Depfile != nil {
		depfile := ninjaEscape(step.Depfile.Absolute())
		fmt.Fprintf(ctx.writer, "  depfile = %s\n", depfile)
	}
	fmt.Fprintf(ctx.writer, "  command = %s\n", step.Cmd)
	if step.Descr != "" {
		fmt.Fprintf(ctx.writer, "  description = %s\n", step.Descr)
	}
	fmt.Fprint(ctx.writer, "\n")
	fmt.Fprintf(ctx.writer, "build %s: r%d %s\n", strings.Join(outs, " "), ctx.nextRuleID, strings.Join(ins, " "))
	fmt.Fprint(ctx.writer, "\n\n")

	ctx.nextRuleID++
}

func (ctx *NinjaContext) Cwd() Path {
	return ctx.cwd
}

type ListTargetsContext struct {
	writer io.Writer
}

func NewListTargetsContext(writer io.Writer) *ListTargetsContext {
	return &ListTargetsContext{writer}
}

func (ctx *ListTargetsContext) Initialize() {}

func (ctx *ListTargetsContext) AddTarget(name string, target interface{}, cwd OutPath) {
	if _, ok := target.(buildable); ok {
		fmt.Fprintln(ctx.writer, name)
	}
}

func (ctx *ListTargetsContext) AddBuildStep(step BuildStep) {}

func (ctx *ListTargetsContext) Cwd() Path {
	return outPath{}
}
