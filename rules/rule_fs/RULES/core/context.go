package core

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type Pool struct {
	Name  string
	Depth uint
}

var ConsolePool Pool = Pool{
	Name: "console",
}

type Context interface {
	AddBuildStep(BuildStep)
	AddBuildStepWithRule(BuildStepWithRule)
	Cwd() OutPath
	BuildChild(c BuildInterface)

	// WithTrace calls the given function, with the given value added
	// to the trace.
	WithTrace(id string, f func(Context))

	// Trace returns the strings in the current trace (most recent last).
	Trace() []string

	// Rules to be collected in a compilation database. Their name must be unique
	RegisterCompDbRule(rule *BuildRule)

	// Obtains a registered rule (if it exists). The boolean is true if it exists
	GetCompDbRule(name string) (*BuildRule, bool)

	registerPool(pool Pool) error
}

// BuildStep represents one build step (i.e., one build command).
// Each BuildStep produces `Out` and `Outs` from `Ins` and `In` by running `Cmd`.
type BuildStep struct {
	Out          OutPath
	Outs         []OutPath
	In           Path
	Ins          []Path
	Depfile      OutPath
	Cmd          string
	Script       string
	Data         string
	DataFileMode os.FileMode
	Descr        string
	Phony        bool
	Pool         *Pool
}

type BuildRule struct {
	Name      string
	Variables map[string]string
}

type BuildStepWithRule struct {
	Outs         []OutPath
	Ins          []Path
	ImplicitDeps []Path
	OrderDeps    []Path
	Variables    map[string]string
	Rule         BuildRule
	Phony        bool
	traces       [][]string
}

type TargetRule struct {
	Target    string
	Ins       []string
	Variables map[string]string
}

func (step *BuildStep) outs() []OutPath {
	if step.Out == nil {
		return step.Outs
	}
	return append(step.Outs, step.Out)
}

func (step *BuildStep) ins() []Path {
	if step.In == nil {
		return step.Ins
	}
	return append(step.Ins, step.In)
}

type BuildInterface interface {
	Build(ctx Context)
}

func AssertIsBuildableTarget(iface BuildInterface) {
	// Do nothing. This function is simply supposed to cause a compilation fail if the
	// type passed does not implement the interface
}

type outputsInterface interface {
	Outputs() []Path
}

type descriptionInterface interface {
	Description() string
}

type runInterface interface {
	Run(args []string) string
}

func AssertIsRunnableTarget(iface runInterface) {
	// Do nothing. This function is simply supposed to cause a compilation fail if the
	// type passed does not implement the interface
}

type reportInterface interface {
	Report(allTargets []interface{}, selectedTargets []interface{}) BuildInterface
}

func AssertIsReportTarget(iface reportInterface) {
	// Do nothing. This function is simply supposed to cause a compilation fail if the
	// type passed does not implement the interface
}

// An interface for runnables that depend on some other set of targets to run, but not to build
// For example, when using a test wrapper.
type extendedRunInterface interface {
	runInterface
	RunDeps() []Path
}

type testInterface interface {
	Test(args []string) string
}

// An interface for tests that depend on some other set of targets to run, but not to build
// For example, when using a test wrapper.
type extendedTestInterface interface {
	testInterface
	TestDeps() []Path
}

func AssertIsTestableTarget(iface testInterface) {
	// Do nothing. This function is simply supposed to cause a compilation fail if the
	// type passed does not implement the interface
}

type CoverageInterface interface {
	Test(args []string) string
	Binaries() []Path
	CoverageData() []OutPath
}

func AssertIsCoverageTarget(iface CoverageInterface) {
	// Do nothing. This function is simply supposed to cause a compilation fail if the
	// type passed does not implement the interface
}

type TranslationUnit struct {
	Source Path
	Object OutPath
	Flags  []string
}

// AnalyzeInterface is an interface for targets compatible with static analysis
type AnalyzeInterface interface {
	TranslationUnits(ctx Context) []TranslationUnit
	AnalysisDeps(ctx Context) []AnalyzeInterface
}

func AssertIsAnalyzeTarget(iface AnalyzeInterface) {
	// Do nothing. This function is simply supposed to cause a compilation fail if the
	// type passed does not implement the interface
}

type context struct {
	cwd              OutPath
	nextRuleID       int
	trace            []string
	leafOutputs      map[Path]bool
	buildSteps       map[string]*BuildStepWithRule
	targetRules      []TargetRule
	compDbBuildRules map[string]*BuildRule
	nestedBuild      bool
	pools            map[string]uint
}

func newContext(vars map[string]interface{}) *context {
	ctx := &context{
		cwd:              outPath{""},
		leafOutputs:      map[Path]bool{},
		buildSteps:       map[string]*BuildStepWithRule{},
		compDbBuildRules: map[string]*BuildRule{},
		nestedBuild:      false,
	}
	return ctx
}

func (ctx *context) WithTrace(id string, f func(Context)) {
	ctx.trace = append(ctx.trace, id)
	defer func() {
		ctx.trace = ctx.trace[:len(ctx.trace)-1]
	}()
	f(ctx)
}

func (ctx *context) Trace() []string {
	// We return a copy of the trace, to avoid mutations.
	return append([]string{}, ctx.trace...)
}

// AddBuildStep adds a build step for the current target.
func (ctx *context) AddBuildStep(step BuildStep) {
	data := ""
	dataFileMode := os.FileMode(0644)
	dataFilePath := ""

	if step.Script != "" {
		if step.Cmd != "" {
			Fatal("cannot specify both Cmd and Script in a build step")
		}
		data = step.Script
		dataFileMode = 0755
	} else if step.Data != "" {
		if step.Cmd != "" {
			Fatal("cannot specify both Cmd and Data in a build step")
		}
		if step.Out == nil || step.Outs != nil {
			Fatal("a single Out is required for Data in a build step")
		}
		data = step.Data
		if step.DataFileMode != 0 {
			dataFileMode = step.DataFileMode
		}
	}

	if data != "" {
		buffer := []byte(data)
		hash := crc32.ChecksumIEEE([]byte(buffer))
		dataFileName := fmt.Sprintf("%08X", hash)
		dataFilePath = path.Join(filepath.Dir(input.OutputDir), "DATA", dataFileName)
		if err := os.MkdirAll(filepath.Dir(dataFilePath), os.ModePerm); err != nil {
			Fatal("Failed to create directory for data files: %s", err)
		}
		if err := ioutil.WriteFile(dataFilePath, buffer, dataFileMode); err != nil {
			Fatal("Failed to write data file: %s", err)
		}
	}

	if step.Script != "" {
		step.Cmd = dataFilePath
	} else if step.Data != "" {
		step.Cmd = fmt.Sprintf("cp %q %q", dataFilePath, step.Out)
	}

	rule := BuildRule{
		Variables: map[string]string{
			"command":     step.Cmd,
			"description": step.Descr,
		},
	}
	if step.Depfile != nil {
		rule.Variables["depfile"] = ninjaEscape(step.Depfile.Absolute())
	}
	if step.Pool != nil {
		if err := ctx.registerPool(*step.Pool); err != nil {
			Fatal("Failed to register ninja pool: %v", err)
		}
		rule.Variables["pool"] = ninjaEscape(step.Pool.Name)
	}

	ctx.AddBuildStepWithRule(BuildStepWithRule{
		Outs:  step.outs(),
		Ins:   step.ins(),
		Rule:  rule,
		Phony: step.Phony,
	})
}

// AddBuildStepWithRule adds a build step for the current target.
func (ctx *context) AddBuildStepWithRule(step BuildStepWithRule) {
	if len(step.Outs) == 0 {
		return
	}

	if prevStep, ok := ctx.buildSteps[step.Outs[0].Absolute()]; ok {
		if err := stepsAreEquivalent(&step, prevStep); err != nil {
			Fatal("Second incompatible build step for output %s: %s", step.Outs[0].Absolute(), err)
		}

		prevStep.traces = append(prevStep.traces, ctx.Trace())

		if !ctx.nestedBuild {
			for _, out := range step.Outs {
				ctx.leafOutputs[out] = true
			}
			for _, in := range step.Ins {
				delete(ctx.leafOutputs, in)
			}
		}
	} else {
		step.traces = append(step.traces, ctx.Trace())

		// Force a copy of step.Outs and step.Ins since changes to these inside build
		// rule code could otherwise corrupt the stored build step.
		step.Outs = append([]OutPath(nil), step.Outs...)
		step.Ins = append([]Path(nil), step.Ins...)

		for _, out := range step.Outs {
			ctx.buildSteps[out.Absolute()] = &step
		}
	}

	if !ctx.nestedBuild {
		for _, out := range step.Outs {
			ctx.leafOutputs[out] = true
		}

		for _, in := range step.Ins {
			delete(ctx.leafOutputs, in)
		}
	}
}

// Cwd returns the build directory of the current target.
func (ctx *context) Cwd() OutPath {
	return ctx.cwd
}

func (ctx *context) BuildChild(c BuildInterface) {
	nb := ctx.nestedBuild
	ctx.nestedBuild = true
	c.Build(ctx)
	ctx.nestedBuild = nb
}

func (ctx *context) registerPool(pool Pool) error {
	if pool.Name == "" {
		return fmt.Errorf("Cannot register a pool with an empty name")
	}

	if pool.Name == "console" {
		// console pool is implicitly defined
		return nil
	}

	if depth, ok := ctx.pools[pool.Name]; ok && depth != pool.Depth {
		return fmt.Errorf("Incompatible pool %q already registered with different depth: %d vs %d", pool.Name, pool.Depth, depth)
	}

	ctx.pools[pool.Name] = pool.Depth
	return nil
}

func (ctx *context) handleTarget(targetPath string, target BuildInterface) {
	currentTarget = targetPath
	ctx.cwd = outPath{path.Dir(targetPath)}
	ctx.leafOutputs = map[Path]bool{}

	ctx.WithTrace("target:"+targetPath, target.Build)

	// Private targets that start with a lower-case letter.
	if !unicode.IsUpper([]rune(path.Base(targetPath))[0]) {
		return
	}

	ninjaOuts := []string{}
	for out := range ctx.leafOutputs {
		ninjaOuts = append(ninjaOuts, ninjaEscape(out.Absolute()))
	}
	sort.Strings(ninjaOuts)

	printOuts := []string{}
	if iface, ok := target.(outputsInterface); ok {
		for _, out := range iface.Outputs() {
			relPath, _ := filepath.Rel(input.WorkingDir, out.Absolute())
			printOuts = append(printOuts, relPath)
		}
	} else {
		for out := range ctx.leafOutputs {
			relPath, _ := filepath.Rel(input.WorkingDir, out.Absolute())
			printOuts = append(printOuts, relPath)
		}
	}
	sort.Strings(printOuts)

	if len(printOuts) == 0 {
		printOuts = []string{"<no outputs produced>"}
	}

	ctx.targetRules = append(ctx.targetRules, TargetRule{
		Target: targetPath,
		Ins:    ninjaOuts,
		Variables: map[string]string{
			"command":     fmt.Sprintf("echo \"%s\"", strings.Join(printOuts, "\\n")),
			"description": fmt.Sprintf("Created %s:", targetPath),
		},
	})

	if runIface, ok := target.(runInterface); ok {
		deps := []string{}
		if extendedRunIface, ok := target.(extendedRunInterface); ok {
			depsPaths := extendedRunIface.RunDeps()
			for _, dep := range depsPaths {
				deps = append(deps, dep.Absolute())
			}
		}

		ctx.targetRules = append(ctx.targetRules, TargetRule{
			Target: fmt.Sprintf("%s#run", targetPath),
			Ins:    append(deps, targetPath),
			Variables: map[string]string{
				"command":     runIface.Run(input.RunArgs),
				"description": fmt.Sprintf("Running %s:", targetPath),
				"pool":        "console",
			},
		})
	}

	if testIface, ok := target.(testInterface); ok {
		deps := []string{}
		if extendedTestIface, ok := target.(extendedTestInterface); ok {
			depsPaths := extendedTestIface.TestDeps()
			for _, dep := range depsPaths {
				deps = append(deps, dep.Absolute())
			}
		}

		ctx.targetRules = append(ctx.targetRules, TargetRule{
			Target: fmt.Sprintf("%s#test", targetPath),
			Ins:    append(deps, targetPath),
			Variables: map[string]string{
				"command":     testIface.Test(input.TestArgs),
				"description": fmt.Sprintf("Testing %s:", targetPath),
				"pool":        "console",
			},
		})
	}
}

func stepsAreEquivalent(a, b *BuildStepWithRule) error {
	if len(a.Ins) != len(b.Ins) {
		return fmt.Errorf("different number of inputs")
	}
	for i := range a.Ins {
		if a.Ins[i] != b.Ins[i] {
			return fmt.Errorf("different input at position %d: %s vs %s", i, a.Ins[i], b.Ins[i])
		}
	}

	if len(a.Outs) != len(b.Outs) {
		return fmt.Errorf("different number of outputs")
	}
	for i := range a.Outs {
		if a.Outs[i] != b.Outs[i] {
			return fmt.Errorf("different output at position %d: %s vs %s", i, a.Outs[i], b.Outs[i])
		}
	}

	if len(a.Variables) != len(b.Variables) {
		return fmt.Errorf("different number of variables")
	}
	for name := range a.Variables {
		if a.Variables[name] != b.Variables[name] {
			return fmt.Errorf("different value for variable '%s' (%s vs %s)", name, a.Variables[name], b.Variables[name])
		}
	}

	if a.Rule.Name != b.Rule.Name {
		return fmt.Errorf("different build rule")
	}
	if len(a.Rule.Variables) != len(b.Rule.Variables) {
		return fmt.Errorf("different number of variables in build rule")
	}
	for name := range a.Rule.Variables {
		if a.Rule.Variables[name] != b.Rule.Variables[name] {
			return fmt.Errorf("different value for of variable '%s' in build rule", name)
		}
	}

	return nil
}

func (ctx *context) ninjaFile() string {
	type kv struct {
		k string
		v string
	}

	sortedBuildRules := func(m map[string]*BuildStepWithRule) []string {
		keys := []string{}
		for key, _ := range m {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return keys
	}

	sortedKvs := func(m map[string]string) []kv {
		keys := []kv{}
		for key, _ := range m {
			keys = append(keys, kv{key, m[key]})
		}
		sort.Slice(keys, func(l, r int) bool { return keys[l].k < keys[r].k })
		return keys
	}

	ninjaFile := &strings.Builder{}
	buildKeys := sortedBuildRules(ctx.buildSteps)

	fmt.Fprintf(ninjaFile, "build __phony__: phony\n\n")

	fmt.Fprintf(ninjaFile, "# pools\n\n")

	for poolName, poolDepth := range ctx.pools {
		fmt.Fprintf(ninjaFile, "pool %s\n", ninjaEscape(poolName))
		fmt.Fprintf(ninjaFile, "  depth = %u\n\n", poolDepth)
	}

	fmt.Fprintf(ninjaFile, "# build rules\n\n")

	seenRules := map[string]bool{}
	i := 0
	for _, key := range buildKeys {
		step := ctx.buildSteps[key]
		if step.Rule.Name == "" {
			step.Rule.Name = fmt.Sprintf("__rule%d", i)
			i++
		}

		if _, ok := seenRules[step.Rule.Name]; ok {
			continue
		}
		seenRules[step.Rule.Name] = true

		fmt.Fprintf(ninjaFile, "rule %s\n", step.Rule.Name)
		for _, kv := range sortedKvs(step.Rule.Variables) {
			fmt.Fprintf(ninjaFile, "  %s = %s\n", kv.k, kv.v)
		}
		fmt.Fprint(ninjaFile, "\n\n")
	}

	fmt.Fprintf(ninjaFile, "# build steps\n\n")

	seenSteps := map[*BuildStepWithRule]bool{}
	for _, key := range buildKeys {
		step := ctx.buildSteps[key]
		if _, ok := seenSteps[step]; ok {
			continue
		}
		seenSteps[step] = true

		outs := []string{}
		for _, out := range step.Outs {
			outs = append(outs, ninjaEscape(out.Absolute()))
		}

		ins := []string{}
		for _, in := range step.Ins {
			ins = append(ins, ninjaEscape(in.Absolute()))
		}
		if step.Phony {
			ins = append(ins, "__phony__")
		}

		orderDeps := []string{}
		for _, in := range step.OrderDeps {
			orderDeps = append(orderDeps, ninjaEscape(in.Absolute()))
		}

		implicitDeps := []string{}
		for _, in := range step.ImplicitDeps {
			implicitDeps = append(implicitDeps, ninjaEscape(in.Absolute()))
		}

		for i, trace := range step.traces {
			fmt.Fprintf(ninjaFile, "# trace: %s\n", strings.Join(trace, " --> "))
			if i == 10 {
				fmt.Fprintf(ninjaFile, "# (skipped %d additional traces)\n", len(step.traces)-10)
				break
			}
		}

		fmt.Fprintf(ninjaFile, "build %s: %s %s | %s || %s\n", strings.Join(outs, " "), step.Rule.Name, strings.Join(ins, " "), strings.Join(implicitDeps, " "), strings.Join(orderDeps, " "))
		for _, kv := range sortedKvs(step.Variables) {
			fmt.Fprintf(ninjaFile, "  %s = %s\n", kv.k, kv.v)
		}
		fmt.Fprint(ninjaFile, "\n\n")
	}

	fmt.Fprintf(ninjaFile, "# targets\n\n")
	for i, target := range ctx.targetRules {
		fmt.Fprintf(ninjaFile, "rule __target%d\n", i)
		for _, kv := range sortedKvs(target.Variables) {
			fmt.Fprintf(ninjaFile, "  %s = %s\n", kv.k, kv.v)
		}
		fmt.Fprintf(ninjaFile, "\n")
		fmt.Fprintf(ninjaFile, "build %s: __target%d %s __phony__\n", target.Target, i, strings.Join(target.Ins, " "))
		fmt.Fprintf(ninjaFile, "\n\n")
	}

	return ninjaFile.String()
}

func (ctx *context) RegisterCompDbRule(rule *BuildRule) {
	ctx.compDbBuildRules[rule.Name] = rule
}

func (ctx *context) GetCompDbRule(name string) (*BuildRule, bool) {
	buildRule, ok := ctx.compDbBuildRules[name]
	return buildRule, ok
}

func ninjaEscape(s string) string {
	return strings.ReplaceAll(s, " ", "$ ")
}
