package core

import (
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
)

const scriptFileMode = 0755

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

type buildsOne interface {
	Build(ctx Context) OutPath
}

type buildsMany interface {
	Build(ctx Context) OutPaths
}

func (ctx *NinjaContext) Initialize() {
	fmt.Fprintf(ctx.writer, "build __phony__: phony\n\n")
}

func (ctx *NinjaContext) AddTarget(name string, target interface{}, cwd OutPath) {
	currentTarget = name
	ctx.cwd = cwd
	outs := OutPaths{}

	if iface, ok := target.(buildsOne); ok {
		outs = append(outs, iface.Build(ctx))
	}

	if iface, ok := target.(buildsMany); ok {
		outs = append(outs, iface.Build(ctx)...)
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

	if step.Script != "" {
		Assert(step.Cmd == "", "cannot specify Cmd and Script in a build step")
		script := []byte(step.Script)
		hash := crc32.ChecksumIEEE([]byte(script))
		scriptFileName := fmt.Sprintf("%08X.sh", hash)
		scriptFilePath := path.Join(buildDir(), "..", scriptFileName)
		err := ioutil.WriteFile(scriptFilePath, script, scriptFileMode)
		if err != nil {
			Fatal("%s", err)
		}
		step.Cmd = scriptFilePath
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
	_, okOne := target.(buildsOne)
	_, okMany := target.(buildsMany)
	if okOne || okMany {
		fmt.Fprintln(ctx.writer, name)
	}
}

func (ctx *ListTargetsContext) AddBuildStep(step BuildStep) {}

func (ctx *ListTargetsContext) Cwd() Path {
	return outPath{}
}
