package hdl

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"dbt-rules/RULES/core"
)

// VlogFlags enables the user to specify additional flags for the 'vlog' command.
var VlogFlags = core.StringFlag{
	Name: "questa-vlog-flags",
	DefaultFn: func() string {
		return "-svinputport=net"
	},
	Description: "Extra flags for the vlog command",
}.Register()

// VcomFlags enables the user to specify additional flags for the 'vcom' command.
var VcomFlags = core.StringFlag{
	Name: "questa-vcom-flags",
	DefaultFn: func() string {
		return ""
	},
	Description: "Extra flags for the vcom command",
}.Register()

// VsimFlags enables the user to specify additional flags for the 'vsim' command.
var VoptFlags = core.StringFlag{
	Name: "questa-vopt-flags",
	DefaultFn: func() string {
		return "-fsmverbose"
	},
	Description: "Extra flags for the vopt command",
}.Register()

// VsimFlags enables the user to specify additional flags for the 'vsim' command.
var VsimFlags = core.StringFlag{
	Name: "questa-vsim-flags",
	DefaultFn: func() string {
		return ""
	},
	Description: "Extra flags for the vsim command",
}.Register()

// Lint enables additional linting information during compilation.
var Lint = core.BoolFlag{
	Name: "questa-lint",
	DefaultFn: func() bool {
		return false
	},
	Description: "Enable additional lint information during compilation",
}.Register()

// Designfile enables the generation of a binary designfile for use with the visualizer
var Designfile = core.BoolFlag{
	Name: "questa-designfile",
	DefaultFn: func() bool {
		return false
	},
	Description: "Enable the creation of a binary designfile database for use with the visualizer",
}.Register()

// Access enables the user to control the accessibility in the compiled design for
// debugging purposes.
var Access = core.StringFlag{
	Name: "questa-access",
	DefaultFn: func() string {
		return "acc"
	},
	Description: "Control access to simulation objects for debugging purposes",
}.Register()

// Coverage enables the user to run the simulation with code coverage.
var Coverage = core.BoolFlag{
	Name: "questa-coverage",
	DefaultFn: func() bool {
		return false
	},
	Description: "Enable code-coverage database generation",
}.Register()

// Coverage enables the user to run the simulation with code coverage.
var DumpQwavedb = core.BoolFlag{
	Name: "questa-dump-qwavedb",
	DefaultFn: func() bool {
		return false
	},
	Description: "Enable waveform dumping to qwavedb file",
}.Register()

var DumpQwavedbScope = core.StringFlag{
	Name: "questa-dump-qwavedb-scope",
	DefaultFn: func() string {
		return "all"
	},
	Description:   "Control the scope of data dumped to qwavedb file",
	AllowedValues: []string{"all", "signals", "assertions", "memory", "queues"},
}.Register()

// paramFlags returns the flags needed to select specific parameters for this rule
func paramFlags(rule Simulation, params string) string {
	cmd := ""
	if params != "" {
		if rule.Params != nil {
			if _, ok := rule.Params[params]; !ok {
				log.Fatal(fmt.Sprintf("parameter set %s not defined!", params))
			}
		} else {
			log.Fatal(fmt.Sprintf("parameter set %s requested, but no parameters sets are defined!", params))
		}
		// Add parameters for all generics into a single string
		for _, name := range rule.SortedParamSet(params) {
			value := rule.Params[params][name]
			cmd += fmt.Sprintf(" -g %s=%s", name, value)
		}
	}
	return cmd
}

// libFlags returns the flags needed to configure the extra libraries for this rule
func libFlags(rule Simulation) string {
	// Holds actual flags
	flags := ""
	// Holds all libraries to avoid duplication
	lib_map := map[string]bool{}

	// get defaults
	for _, lib := range append(strings.Split(SimulatorLibSearch.Value(), " "), rule.Libs...) {
		if lib != "" {
			if _, ok := lib_map[lib]; !ok {
				lib_map[lib] = true
				flags += " -L " + lib
			}
		}
	}

	return flags
}

// Construct a string of +incdir+%s arguments from a list of directories
func incDirFlags(incs []core.Path) string {
	cmd := ""
	seen_incs := make(map[string]struct{})
	for _, inc := range incs {
		inc_path := path.Dir(inc.Absolute())
		if _, ok := seen_incs[inc_path]; !ok {
			cmd += fmt.Sprintf(" +incdir+%s", inc_path)
			seen_incs[inc_path] = struct{}{}
		}
	}
	return cmd
}

func removeDuplicateFiles(deps []core.Path) []core.Path {
	// Remove duplicates
	set := make(map[string]core.Path)
	for _, dep := range deps {
		set[dep.String()] = dep
	}

	// Convert back to string list
	paths := make([]core.Path, len(set))
	i := 0
	for _, value := range set {
		paths[i] = value
		i++
	}
	return paths
}

// verbosityLevelToFlag takes a verbosity level of none, low, medium, high or all and
// converts it to the corresponding DVM_ level.
func verbosityLevelToFlag(level string) (string, bool) {
	var verbosity_flag string
	var print_output bool
	switch level {
	case "none":
		verbosity_flag = " +verbosity=DVM_VERB_NONE"
		print_output = false
	case "low":
		verbosity_flag = " +verbosity=DVM_VERB_LOW"
		print_output = true
	case "medium":
		verbosity_flag = " +verbosity=DVM_VERB_MED"
		print_output = true
	case "high":
		verbosity_flag = " +verbosity=DVM_VERB_HIGH"
		print_output = true
	case "all":
		verbosity_flag = " +verbosity=DVM_VERB_ALL"
		print_output = true
	default:
		log.Fatal(fmt.Sprintf("invalid verbosity flag '%s', only 'low', 'medium',"+
			" 'high', 'all'  or 'none' allowed!", level))
	}

	return verbosity_flag, print_output
}

// rules holds a map of all defined rules to prevent defining the same rule
// multiple times.
var rules = make(map[string]bool)

// common_flags holds common flags used for the 'vlog', 'vcom', and 'vopt' commands.
const common_flags = "-nologo -quiet -work work"

type Target struct {
	Name    string
	LogFile core.OutPath
	Params  string
}

// Parameters of the do-file
type doFileParams struct {
	WaveformInit string
	DumpVcd      bool
	DumpVcdFile  string
	CovFiles     string
}

// Do-file template
const do_file_template = `
proc reload {} {
	global target
	vsim -work work $target
	{{ if .WaveformInit }}
		source {{ .WaveformInit }}
	{{ end }}

}

set StdArithNoWarnings 1
set NumericStdNoWarnings 1

{{ if .WaveformInit }}
if [info exists gui] {
	run 1
	catch { source {{ .WaveformInit }} }
	assertion fail -action break
}
{{ end }}

if [info exists from] {
	run $from
}

{{ if .DumpVcd }}
vcd file {{ .DumpVcdFile }}
vcd add -r *
{{ end }}

if [info exists to] {
	run @$to
} else {
	run -all
}

{{ if .DumpVcd }}
vcd flush
{{ end }}

if [info exists coverage] {
	# Create coverage database
	coverage save -assert -directive -cvg -codeall -testname $testcase $coverage_db.ucdb
	# Optionally merge coverage databases
	if {$main_coverage_db != $coverage_db} {
		puts "Writing merged coverage database to [pwd]/$main_coverage_db.ucdb"
		vcover merge -testassociated -output $main_coverage_db.ucdb $main_coverage_db.ucdb $coverage_db.ucdb
	}
	# Create HTML coverage report
	vcover report -html -output ${main_coverage_db}_covhtml \
		-testdetails -details -assert -directive -cvg -codeAll $main_coverage_db.ucdb
	# Create textual code coverage report
	{{ if .CovFiles }}
	vcover report -output ${main_coverage_db}_covcode.txt -srcfile={{ .CovFiles }}\
		-codeAll $main_coverage_db.ucdb
	{{ else }}
	vcover report -output ${main_coverage_db}_covcode.txt\
		-codeAll $main_coverage_db.ucdb
	{{ end }}
	# Create textual assertion coverage report
	puts "Writing coverage report to [pwd]/${main_coverage_db}_cover.txt"
	vcover report -output ${main_coverage_db}_cover.txt -flat -directive -cvg $main_coverage_db.ucdb
	# Create textural assertion report
	puts "Writing assertion report to [pwd]/${main_coverage_db}_cover.txt"
	vcover report -output ${main_coverage_db}_assert.txt -flat -assert $main_coverage_db.ucdb
}

if ![info exists gui] {
  quit -code [expr [coverage attribute -name TESTSTATUS -concise] > 1]
}
`

func createModelsimIni(ctx core.Context, rule Simulation, deps []core.Path) []core.Path {
	questa_lib := core.BuildPath("questa_lib")
	ctx.AddBuildStep(core.BuildStep{
		Out:   questa_lib,
		Cmd:   fmt.Sprintf("mkdir %q", questa_lib.Absolute()),
		Descr: fmt.Sprintf("mkdir: %q", questa_lib.Absolute()),
	})
	deps = append(deps, questa_lib)

	modelsim_ini := core.BuildPath("modelsim.ini")

	// Standard libraries needed by Vivado
	cmds := []string{
		"vlib questa_lib/work",
		"vlib questa_lib/msim",
		"vlib questa_lib/msim/xil_defaultlib",
		"vmap work questa_lib/work",
		"vmap xil_defaultlib questa_lib/msim/xil_defaultlib",
	}

	if SimulatorLibDir.Value() != "" {
		cmds = append(cmds, fmt.Sprintf(
			"if [ -d \"%s\" ]; then for lib in $$(find %s -mindepth 1 -maxdepth 1 -type d); do vmap $$(basename $$lib) $$lib; done; fi",
			SimulatorLibDir.Value(), SimulatorLibDir.Value()))
	}

	ctx.AddBuildStep(core.BuildStep{
		In:    questa_lib,
		Out:   modelsim_ini,
		Cmd:   strings.Join(cmds, " && "),
		Descr: fmt.Sprintf("vmap: %s", modelsim_ini.Absolute()),
	})
	deps = append(deps, modelsim_ini)

	return deps
}

// Create a command for running vlog on a file; the file is not part of the returned command
func vlogCmd(ctx core.Context, rule Simulation, incs []core.Path, flags FlagMap) string {
	cmd := "vlog " + common_flags
	cmd += libFlags(rule)
	cmd += " +incdir+" + core.SourcePath("").String()
	cmd += incDirFlags(incs)

	if flags != nil {
		if vlog_flags, ok := flags["vlog"]; ok {
			cmd += " " + vlog_flags
		}
	}

	cmd += "  -define SIMULATION"
	for _, key := range sortedStringKeys(rule.Defines) {
		cmd += " -define " + key
		if rule.Defines[key] != "" {
			cmd += fmt.Sprintf("=%s", rule.Defines[key])
		}
	}

	return cmd
}

// Create a command for running vcom on a file; the file is not part of the returned command
func vcomCmd(ctx core.Context, rule Simulation, flags FlagMap) string {
	cmd := "vcom " + common_flags

	if flags != nil {
		if vcom_flags, ok := flags["vcom"]; ok {
			cmd += " " + vcom_flags
		}
	}

	return cmd
}

// compileSrcs compiles a list of sources using the specified context ctx, rule,
// dependencies and include paths. It returns the resulting dependencies and include paths
// that result from compiling the source files.
func compileSrcs(ctx core.Context, rule Simulation,
	deps []core.Path, incs []core.Path, srcs []core.Path, flags FlagMap) ([]core.Path, []core.Path) {
	for _, src := range srcs {
		// log will point to the log file to be generated when compiling the code
		log := core.BuildPath(src.Relative()).WithSuffix(".log")
		// Command will be updated to compile the source code
		cmd := ""
		// Tool will indicate the used tool
		tool := ""

		if IsRtl(src.String()) {
			// tool will point to the tool to execute (also used for logging below)
			if IsVerilog(src.String()) {
				tool = "vlog"
				cmd += vlogCmd(ctx, rule, incs, flags)
			} else if IsVhdl(src.String()) {
				tool = "vcom"
				cmd += vcomCmd(ctx, rule, flags)
			}

			if Lint.Value() {
				cmd += " -lint"
			}

			cmd += " -l " + log.String() + " " + src.String()

		} else if IsXilinxIpCheckpoint(src.String()) {
			tool = "vsim"
			src = ExportXilinxIpCheckpoint(ctx, rule, src, rule.Defines, flags)
			cmd = fmt.Sprintf("vsim -batch -do \"set t [exec date -R -r modelsim.ini]\" -do %s -do \"exec touch -d \\$$t modelsim.ini\" -do exit -logfile %s", src.String(), log.String())
		} else if IsHeader(src.String()) {
			// Header files are added to the list to be able to set include directories correctly
			incs = append(incs, src)
		}

		// Just add the file to the dependencies of the next one (including header files)
		deps = append(deps, src)

		if cmd != "" {
			// Remove the log file if the command fails to ensure we can recompile it
			cmd += " || { rm " + log.String() + " && exit 1; }"

			// If we already have a rule for this file, skip it.
			if !rules[log.String()] {
				// Add the compilation command as a build step with the log file as the
				// generated output
				ctx.AddBuildStep(core.BuildStep{
					Out:   log,
					Ins:   deps,
					Cmd:   cmd,
					Descr: fmt.Sprintf("%s: %s", tool, src.Absolute()),
				})

				// Note down the created rule
				rules[log.String()] = true
			}

			// Add the log file to the dependencies of the next files
			deps = append(deps, log)
		}
	}

	return deps, incs
}

// compileBlockDesign exports and compiles a BlockDesign
func compileBlockDesign(ctx core.Context, rule Simulation, ip BlockDesign, deps []core.Path, flags FlagMap) []core.Path {
	log := ctx.Cwd().WithSuffix("/" + ip.Name + ".log")

	if !rules[log.String()] {
		// Merge options
		for tool, flag := range ip.Flags() {
			flags[tool] = flag
		}

		do := ExportBlockDesign(ctx, ip, rule.Defines, flags)
		ctx.AddBuildStep(core.BuildStep{
			Out:   log,
			In:    do,
			Cmd:   fmt.Sprintf("vsim -batch -do \"set t [exec date -R -r modelsim.ini]\" -do %s -do \"exec touch -d \\$$t modelsim.ini\" -do exit -logfile %s", do.String(), log.String()),
			Descr: fmt.Sprintf("vsim: %s", do.Absolute()),
		})

		// Note down the created rule
		rules[log.String()] = true
	}

	return append(deps, log)
}

// compileIp compiles the IP dependencies and the source files of a Library
func compileIp(ctx core.Context, rule Simulation, ip Ip,
	deps []core.Path, incs []core.Path, flags FlagMap) ([]core.Path, []core.Path) {

	// Merge tool options
	for tool, flag := range ip.Flags() {
		if val, ok := flags[tool]; !ok {
			flags[tool] = flag
		} else {
			flags[tool] = val + " " + flag
		}
	}

	// Update board and part
	if v, ok := ip.(Fpga); ok {
		flags["part"] = v.Part
		flags["board"] = v.Board
	}

	// Compile Ips
	for _, sub_ip := range ip.Ips() {
		deps, incs = compileIp(ctx, rule, sub_ip, deps, incs, flags)
	}
	// and local sources
	deps, incs = compileSrcs(ctx, rule, deps, incs, ip.Sources(), flags)

	if v, ok := ip.(BlockDesign); ok {
		deps = compileBlockDesign(ctx, rule, v, deps, flags)
	}

	return deps, incs
}

// compile compiles the IP dependencies and source files of a simulation rule.
func compile(ctx core.Context, rule Simulation) []core.Path {
	incs := []core.Path{}
	deps := []core.Path{}

	// Collect aditional tool flags from rule
	flags := FlagMap{}
	if rule.ToolFlags != nil {
		flags = rule.ToolFlags
	}

	// Collect remaining tool flags for the vlog and vcom tools
	if val, ok := flags["vlog"]; !ok {
		flags["vlog"] = VlogFlags.Value()
	} else {
		flags["vlog"] = val + " " + VlogFlags.Value()
	}

	if val, ok := flags["vcom"]; !ok {
		flags["vcom"] = VcomFlags.Value()
	} else {
		flags["vcom"] = val + " " + VlogFlags.Value()
	}

	deps = createModelsimIni(ctx, rule, deps)

	for _, ip := range rule.Ips {
		deps, incs = compileIp(ctx, rule, ip, deps, incs, flags)
	}
	deps, incs = compileSrcs(ctx, rule, deps, incs, rule.Srcs, flags)

	return deps
}

// optimize creates and optimized version of the design optionally including
// coverage recording functionality. The optimized design unit can then conveniently
// be simulated using 'vsim'.
func optimize(ctx core.Context, rule Simulation, deps []core.Path) {
	if rule.Top != "" && len(rule.Tops) > 0 {
		log.Fatal(fmt.Sprintf("only one of Top or Tops allowed!"))
	}

	// Default for compatibility
	tops := []string{"board"}
	if rule.Top != "" {
		tops = []string{rule.Top}
	} else if len(rule.Tops) > 0 {
		tops = rule.Tops
	}

	log_file_suffix := "vopt.log"

	cover_flag := ""
	if Coverage.Value() {
		cover_flag = "+cover"
	}

	// Will hold all targets for optimization
	targets := []Target{}

	if rule.Params != nil {
		for params_name := range rule.Params {
			targets = append(targets, Target{
				Name:    rule.Target(params_name, Coverage.Value()),
				LogFile: rule.Path().WithSuffix("/" + params_name + "_" + log_file_suffix),
				Params:  params_name,
			})
		}
	} else {
		targets = append(targets, Target{
			Name:    rule.Target("", Coverage.Value()),
			LogFile: rule.Path().WithSuffix("/" + log_file_suffix),
			Params:  "",
		})
	}

	// Generate access flag
	access_flag := ""
	switch Access.Value() {
	case "debug":
		access_flag = "-debug"
	case "livesim":
		access_flag = "-debug,livesim"
	case "acc":
		access_flag = "+acc"
	case "":
	default:
		access_flag = fmt.Sprintf("+acc=%s", Access.Value())
	}

	for _, target := range targets {
		// Skip if we already have a rule
		if rules[target.LogFile.String()] {
			continue
		}

		// Generate designfile flag
		designfile_flag := ""
		if Designfile.Value() {
			design_file := "design"
			if target.Params != "" {
				design_file = design_file + "_" + target.Params
			}

			designfile_flag = "-designfile " + rule.Path().WithSuffix("/"+design_file+".bin").String()
		}

		cmd := "vopt " + common_flags
		cmd += " " + VoptFlags.Value()
		cmd += " " + cover_flag
		cmd += " " + access_flag
		cmd += " " + designfile_flag
		cmd += libFlags(rule)
		cmd += paramFlags(rule, target.Params)

		// Add any extra flags specified with the rule
		if rule.ToolFlags != nil {
			if vopt_flags, ok := rule.ToolFlags["vopt"]; ok {
				cmd += " " + vopt_flags
			}
		}

		cmd += " -l " + target.LogFile.String()
		cmd += " " + strings.Join(tops, " ")
		cmd += " -o " + target.Name

		if rule.TestCaseGenerator != nil {
			deps = append(deps, rule.TestCaseGenerator)
		}

		// Add the rule to run 'vopt'.
		ctx.AddBuildStep(core.BuildStep{
			Out:   target.LogFile,
			Ins:   deps,
			Cmd:   cmd,
			Descr: fmt.Sprintf("vopt: %s -o %s", strings.Join(tops, " "), target.Name),
		})

		// Note that we created this rule
		rules[target.LogFile.String()] = true
	}
}

// Create a simulation script
func doFile(ctx core.Context, rule Simulation) {
	// Do-file script
	params := doFileParams{
		DumpVcd:     DumpVcd.Value(),
		DumpVcdFile: rule.Path().WithSuffix("/waves.vcd.gz").String(),
		CovFiles:    strings.Join(rule.ReportCovFiles(), "+"),
	}

	if rule.WaveformInit != nil {
		params.WaveformInit = rule.WaveformInit.String()
	}

	doFile := rule.Path().WithSuffix("/" + "vsim.do")
	ctx.AddBuildStep(core.BuildStep{
		Out:   doFile,
		Data:  core.CompileTemplate(do_file_template, "do", params),
		Descr: fmt.Sprintf("template: %s", doFile.Absolute()),
	})
}

// BuildQuesta will compile and optimize the source and IPs associated with the given
// rule.
func BuildQuesta(ctx core.Context, rule Simulation) {
	// compile the code
	deps := compile(ctx, rule)

	// optimize the code
	optimize(ctx, rule, deps)

	// Create script
	doFile(ctx, rule)
}

// vsimCmd will create a command for starting 'vsim' on the compiled and optimized design with flags
// set in accordance with what is specified on the command line.
func vsimCmd(rule Simulation, args []string, gui bool, testcase string, params string) string {
	// Prefix the vsim command with this
	cmd_preamble := ""

	// Default log file
	log_file_suffix := "vsim.log"
	if testcase != "" {
		log_file_suffix = testcase + "_" + log_file_suffix
	}
	if params != "" {
		log_file_suffix = params + "_" + log_file_suffix
	}
	log_file := rule.Path().WithSuffix("/" + log_file_suffix)

	// Script to execute
	do_file := rule.Path().WithSuffix("/" + "vsim.do")

	// Collect do-files and commands here
	var do_flags []string

	// Default flag values
	vsim_flags := " -onfinish final -l " + log_file.String() + libFlags(rule)

	seed_flag := " -sv_seed random"
	verbosity_flag := " +verbosity=DVM_VERB_NONE"
	mode_flag := " -batch -quiet"
	plusargs_flag := ""

	// Default database name for simulation
	target := rule.Target(params, Coverage.Value())

	// Enable coverage in simulator
	coverage_flag := ""
	if Coverage.Value() {
		coverage_flag = " -coverage -assertdebug"
		do_flags = append(do_flags, "\"set coverage 1\"")
	}

	// Enable qwavedb dumping
	qwavedb_flag := ""
	if DumpQwavedb.Value() {
		qwavedb_flag = " -qwavedb="
		switch DumpQwavedbScope.Value() {
		case "signals":
			qwavedb_flag += "+signal"
		case "assertions":
			qwavedb_flag += "+signal+assertions=pass,atv"
		case "memory":
			qwavedb_flag += "+signal+assertions=pass,atv+memory"
		case "queues":
			qwavedb_flag += "+signal+assertions=pass,atv+memory+queues"
		case "all":
			qwavedb_flag += "+signal+assertions=pass,atv+memory+queues+class+classmemory+classdynarray"
		}
		qwavedb_file := "waves"
		if params != "" {
			qwavedb_file = qwavedb_file + "_" + params
		}
		qwavedb_flag += "+wavefile=" + rule.Path().WithSuffix("/"+qwavedb_file+".db").String()
	}

	// Determine the names of the coverage databases, this one will hold merged
	// data from multiple testcases
	main_coverage_db := rule.Name

	// This will be the name of the database created by the current run
	coverage_db := rule.Name

	// Turn off output unless verbosity is activated
	print_output := false

	// Parse additional arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "-seed=") {
			// Define simulator seed
			var seed int64
			if _, err := fmt.Sscanf(arg, "-seed=%d", &seed); err == nil {
				seed_flag = fmt.Sprintf(" -sv_seed %d", seed)
			} else {
				log.Fatal("-seed expects an integer argument!")
			}
		} else if strings.HasPrefix(arg, "-verbosity=") {
			// Define verbosity level
			var level string
			if _, err := fmt.Sscanf(arg, "-verbosity=%s", &level); err == nil {
				verbosity_flag, print_output = verbosityLevelToFlag(level)
			} else {
				log.Fatal("-verbosity expects an argument of 'low', 'medium', 'high' or 'none'!")
			}
		} else if strings.HasPrefix(arg, "-from=") {
			// Define how long to run
			var from string
			if _, err := fmt.Sscanf(arg, "-from=%s", &from); err == nil {
				do_flags = append(do_flags, fmt.Sprintf("\"set from %s\"", from))
			} else {
				log.Fatal("-from expects an argument of '<timesteps>[<time units>]'!")
			}
		} else if strings.HasPrefix(arg, "-to=") {
			// Define how long to run
			var to string
			if _, err := fmt.Sscanf(arg, "-to=%s", &to); err == nil {
				do_flags = append(do_flags, fmt.Sprintf("\"set to %s\"", to))
			} else {
				log.Fatal("-to expects an argument of '<timesteps>[<time units>]'!")
			}
		} else if strings.HasPrefix(arg, "+") {
			// All '+' arguments go directly to the simulator
			plusargs_flag = plusargs_flag + " " + arg
		}
	}

	// Create optional command preamble
	cmd_preamble, testcase = Preamble(rule, testcase)

	cmd_echo := ""
	if rule.Params != nil && params != "" {
		// Update coverage database name based on parameters. Since we cannot merge
		// different parameter sets, we have to make a dedicated main database
		// for this parameter set.
		main_coverage_db = main_coverage_db + "_" + params
		coverage_db = coverage_db + "_" + params
		cmd_echo = "Testcase " + params

		// Update with testcase if specified
		if testcase != "" {
			coverage_db = coverage_db + "_" + testcase
			cmd_echo = cmd_echo + "/" + testcase + ":"
			testcase = params + "_" + testcase
		} else {
			cmd_echo = cmd_echo + ":"
			testcase = params
		}
	} else {
		// Update coverage database name with testcase alone, main database stays
		// the same
		if testcase != "" {
			coverage_db = coverage_db + "_" + testcase
			cmd_echo = "Testcase " + testcase + ":"
		} else {
			testcase = "default"
		}
	}

	do_flags = append(do_flags, fmt.Sprintf("\"set target %s\"", target))
	do_flags = append(do_flags, fmt.Sprintf("\"set testcase %s\"", testcase))
	do_flags = append(do_flags, fmt.Sprintf("\"set main_coverage_db %s\"", main_coverage_db))
	do_flags = append(do_flags, fmt.Sprintf("\"set coverage_db %s\"", coverage_db))

	cmd_postamble := ""
	cmd_pass := "PASS"
	cmd_fail := "FAIL"
	if gui {
		do_flags = append(do_flags, "\"set gui 1\"")
		if Designfile.Value() {
			design_file := "design"
			if params != "" {
				design_file += "_" + params
			}
			mode_flag = " -visualizer=+designfile=" + rule.Path().WithSuffix("/"+design_file+".bin").String()
		} else {
			mode_flag = " -gui"
		}
	}

	if !print_output && !gui {
		mode_flag = mode_flag + " -nostdout"
	}

	if Coverage.Value() {
		cmd_pass = cmd_pass + fmt.Sprintf(" Coverage: $$(pwd)/%s.ucdb", main_coverage_db)
		cmd_fail = cmd_fail + fmt.Sprintf(" Coverage: $$(pwd)/%s.ucdb", main_coverage_db)
	}

	cmd_newline := ":"
	if cmd_echo != "" {
		cmd_newline = "echo"
	}

	if !print_output {
		cmd_postamble = fmt.Sprintf("|| { %s; cat %s; echo %s; exit 1; }", cmd_newline, log_file.String(), cmd_fail)
	}

	vsim_flags = vsim_flags + mode_flag + seed_flag + coverage_flag + qwavedb_flag +
		verbosity_flag + plusargs_flag + " " + VsimFlags.Value()

	// Add any extra flags specified with the rule
	if rule.ToolFlags != nil {
		if extra_flags, ok := rule.ToolFlags["vsim"]; ok {
			vsim_flags = vsim_flags + " " + extra_flags
		}
	}

	for _, do_flag := range do_flags {
		vsim_flags = vsim_flags + " -do " + do_flag
	}

	// Add the file as the last argument
	vsim_flags = vsim_flags + " -do " + do_file.String()

	cmd := fmt.Sprintf("{ echo -n %s && vsim %s -work work %s && echo %s; }", cmd_echo, vsim_flags, target, cmd_pass)
	if cmd_preamble == "" {
		cmd += " " + cmd_postamble
	} else {
		cmd = "{ { " + cmd_preamble + " } && " + cmd + " } " + cmd_postamble
	}

	// Wrap command in another layer of {} to enable chaining
	cmd = "{ " + cmd + " }"

	return cmd
}

// simulateQuesta will create a command to start 'vsim' on the compiled design
// with flags set in accordance with what is specified on the command line. It will
// optionally build a chain of commands in case the rule has parameters, but
// no parameters are specified on the command line
func simulateQuesta(rule Simulation, args []string, gui bool) string {
	// Optional testcase goes here
	testcases := []string{}

	// Optional parameter set goes here
	params := []string{}

	// Parse additional arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "-testcases=") && rule.TestCaseGenerator != nil {
			var testcases_arg string
			if _, err := fmt.Sscanf(arg, "-testcases=%s", &testcases_arg); err != nil {
				log.Fatal(fmt.Sprintf("-testcases expects a string argument!"))
			}
			testcases = append(testcases, strings.Split(testcases_arg, ",")...)
		} else if strings.HasPrefix(arg, "-params=") && rule.Params != nil {
			var params_arg string
			if _, err := fmt.Sscanf(arg, "-params=%s", &params_arg); err != nil {
				log.Fatal(fmt.Sprintf("-params expects a string argument!"))
			} else {
				for _, param := range strings.Split(params_arg, ",") {
					if _, ok := rule.Params[param]; ok {
						params = append(params, param)
					}
				}
			}
		}
	}

	// If no parameters have been specified, simulate them all
	if rule.Params != nil && len(params) == 0 {
		for _, key := range rule.SortedParams() {
			params = append(params, key)
		}
	} else if len(params) == 0 {
		params = append(params, "")
	}

	// If no testcase has been specified, simulate them all
	if rule.TestCaseGenerator != nil && rule.TestCasesDir != nil && len(testcases) == 0 {
		// Loop through all defined testcases in directory
		if items, err := os.ReadDir(rule.TestCasesDir.String()); err == nil {
			for _, item := range items {
				testcases = append(testcases, item.Name())
			}
		} else {
			log.Fatal(err)
		}
	} else if len(testcases) == 0 {
		testcases = append(testcases, "")
	}

	// Final command
	cmd := "{ :; }"

	// Loop for all parameter sets
	for i := range params {
		// Loop for all test cases
		for j := range testcases {
			cmd += " && " + vsimCmd(rule, args, gui, testcases[j], params[i])
			// Only one testcase allowed in GUI mode
			if gui {
				break
			}
		}
		// Only one parameter set allowed in gui mode
		if gui {
			break
		}
	}

	return cmd
}

// Run will build the design and run a simulation in GUI mode.
func RunQuesta(rule Simulation, args []string) string {
	return simulateQuesta(rule, args, true)
}

// Test will build the design and run a simulation in batch mode.
func TestQuesta(rule Simulation, args []string) string {
	return simulateQuesta(rule, args, false)
}
