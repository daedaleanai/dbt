package core

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const scriptFileMode = 0755

type Context interface {
	AddBuildStep(BuildStep)
	BuildPath(string) OutPath
	Cwd() OutPath
	SourcePath(string) Path
}

// BuildStep represents one build step (i.e., one build command).
// Each BuildStep produces `Out` and `Outs` from `Ins` and `In` by running `Cmd`.
type BuildStep struct {
	Out     OutPath
	Outs    []OutPath
	In      Path
	Ins     []Path
	Depfile OutPath
	Cmd     string
	Script  string
	Descr   string
}

type buildInterface interface {
	Build(ctx Context)
}

type outputInterface interface {
	Output() OutPath
}

type outputsInterface interface {
	Outputs() []OutPath
}

type descriptionInterface interface {
	Description() string
}

type context struct {
	skipNinjaFile bool

	cwd         OutPath
	leafOutputs map[string]struct{}

	nextRuleID int

	ninjaFile strings.Builder
	targets   map[string]string
}

func newContext(skipNinja bool) *context {
	ctx := &context{}
	ctx.skipNinjaFile = skipNinja
	ctx.targets = map[string]string{}

	if !ctx.skipNinjaFile {
		fmt.Fprintf(&ctx.ninjaFile, "build __phony__: phony\n\n")
	}

	return ctx
}

func (ctx *context) addTarget(cwd OutPath, name string, target interface{}) {
	currentTarget = name
	ctx.cwd = cwd
	ctx.leafOutputs = map[string]struct{}{}

	iface, ok := target.(buildInterface)
	if !ok {
		return
	}
	if !ctx.skipNinjaFile {
		iface.Build(ctx)
	}

	if iface, ok := target.(descriptionInterface); ok {
		ctx.targets[name] = iface.Description()
	} else {
		ctx.targets[name] = ""
	}

	ninjaOuts := []string{}
	for out := range ctx.leafOutputs {
		ninjaOuts = append(ninjaOuts, out)
	}
	sort.Strings(ninjaOuts)
	if len(ninjaOuts) == 0 {
		return
	}

	printOuts := []string{}
	if iface, ok := target.(outputsInterface); ok {
		for _, out := range iface.Outputs() {
			rel, _ := filepath.Rel(workingDir(), out.Absolute())
			printOuts = append(printOuts, rel)
		}
	}
	if iface, ok := target.(outputInterface); ok {
		rel, _ := filepath.Rel(workingDir(), iface.Output().Absolute())
		printOuts = append(printOuts, rel)
	}

	fmt.Fprintf(&ctx.ninjaFile, "rule r%d\n", ctx.nextRuleID)
	fmt.Fprintf(&ctx.ninjaFile, "  command = echo \"%s\"\n", strings.Join(printOuts, "\\n"))
	fmt.Fprintf(&ctx.ninjaFile, "  description = Created %s:", name)
	fmt.Fprintf(&ctx.ninjaFile, "\n")
	fmt.Fprintf(&ctx.ninjaFile, "build %s: r%d %s __phony__\n", name, ctx.nextRuleID, strings.Join(ninjaOuts, " "))
	fmt.Fprintf(&ctx.ninjaFile, "\n")
	fmt.Fprintf(&ctx.ninjaFile, "\n")

	ctx.nextRuleID++
}

func (ctx *context) AddBuildStep(step BuildStep) {
	outs := []string{}
	for _, out := range step.Outs {
		ninjaOut := ninjaEscape(out.Absolute())
		outs = append(outs)
		ctx.leafOutputs[ninjaOut] = struct{}{}
	}
	if step.Out != nil {
		ninjaOut := ninjaEscape(step.Out.Absolute())
		outs = append(outs, ninjaEscape(step.Out.Absolute()))
		ctx.leafOutputs[ninjaOut] = struct{}{}
	}
	if len(outs) == 0 {
		return
	}

	ins := []string{}
	for _, in := range step.Ins {
		ninjaIn := ninjaEscape(in.Absolute())
		ins = append(ins, ninjaIn)
		delete(ctx.leafOutputs, ninjaIn)
	}
	if step.In != nil {
		ninjaIn := ninjaEscape(step.In.Absolute())
		ins = append(ins, ninjaIn)
		delete(ctx.leafOutputs, ninjaIn)
	}

	if step.Script != "" {
		Assert(step.Cmd == "", "cannot specify Cmd and Script in a build step")
		script := []byte(step.Script)
		hash := crc32.ChecksumIEEE([]byte(script))
		scriptFileName := fmt.Sprintf("%08X.sh", hash)
		scriptFilePath := path.Join(buildDir(), "..", scriptFileName)
		err := ioutil.WriteFile(scriptFilePath, script, scriptFileMode)
		Assert(err == nil, "%s", err)
		step.Cmd = scriptFilePath
	}

	fmt.Fprintf(&ctx.ninjaFile, "rule r%d\n", ctx.nextRuleID)
	if step.Depfile != nil {
		depfile := ninjaEscape(step.Depfile.Absolute())
		fmt.Fprintf(&ctx.ninjaFile, "  depfile = %s\n", depfile)
	}
	fmt.Fprintf(&ctx.ninjaFile, "  command = %s\n", step.Cmd)
	if step.Descr != "" {
		fmt.Fprintf(&ctx.ninjaFile, "  description = %s\n", step.Descr)
	}
	fmt.Fprint(&ctx.ninjaFile, "\n")
	fmt.Fprintf(&ctx.ninjaFile, "build %s: r%d %s\n", strings.Join(outs, " "), ctx.nextRuleID, strings.Join(ins, " "))
	fmt.Fprint(&ctx.ninjaFile, "\n\n")

	ctx.nextRuleID++
}

// BuildPath returns a path relative to the build directory.
func (ctx *context) BuildPath(p string) OutPath {
	return NewOutPath(p)
}

// Cwd returns the build directory of the current target.
func (ctx *context) Cwd() OutPath {
	return ctx.cwd
}

// SourcePath returns a path relative to the source directory.
func (ctx *context) SourcePath(p string) Path {
	return NewInPath(p)
}

func ninjaEscape(s string) string {
	return strings.ReplaceAll(s, " ", "$ ")
}
