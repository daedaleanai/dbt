package core

import (
	"fmt"
	"hash/crc32"
<<<<<<< HEAD
=======
	"io"
>>>>>>> main
	"io/ioutil"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const scriptFileMode = 0755

type Context interface {
	AddBuildStep(BuildStep)
<<<<<<< HEAD
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
=======
	Cwd() Path
}

type NinjaContext struct {
	writer     io.Writer
	nextRuleID int
	cwd        OutPath
}

func NewNinjaContext(writer io.Writer) *NinjaContext {
	return &NinjaContext{writer, 0, outPath{}}
>>>>>>> main
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

<<<<<<< HEAD
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
=======
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
>>>>>>> main
	}

	ninjaOuts := []string{}
	for out := range ctx.leafOutputs {
		ninjaOuts = append(ninjaOuts, out)
	}
	sort.Strings(ninjaOuts)
	if len(ninjaOuts) == 0 {
		return
	}

<<<<<<< HEAD
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
=======
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
>>>>>>> main

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

<<<<<<< HEAD
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
=======
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
>>>>>>> main

	ctx.nextRuleID++
}

<<<<<<< HEAD
// BuildPath returns a path relative to the build directory.
func (ctx *context) BuildPath(p string) OutPath {
	return NewOutPath(p)
=======
func (ctx *NinjaContext) Cwd() Path {
	return ctx.cwd
}

type ListTargetsContext struct {
	writer io.Writer
}

func NewListTargetsContext(writer io.Writer) *ListTargetsContext {
	return &ListTargetsContext{writer}
>>>>>>> main
}

// Cwd returns the build directory of the current target.
func (ctx *context) Cwd() OutPath {
	return ctx.cwd
}

<<<<<<< HEAD
// SourcePath returns a path relative to the source directory.
func (ctx *context) SourcePath(p string) Path {
	return NewInPath(p)
}

func ninjaEscape(s string) string {
	return strings.ReplaceAll(s, " ", "$ ")
=======
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
>>>>>>> main
}
