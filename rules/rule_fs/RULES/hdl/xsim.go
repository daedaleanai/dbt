package hdl

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"dbt-rules/RULES/core"
)

// XsimDumpWdb enables the user to dump all signals to a .wdb file
var XsimDumpWdb = core.BoolFlag{
	Name: "xsim-dump-wdb",
	DefaultFn: func() bool {
		return false
	},
	Description: "Enable output of signals to a WDB file",
}.Register()

// XelabFlags enables the user to specify additional flags for the 'xelab' command.
var XelabFlags = core.StringFlag{
	Name: "xsim-xelab-flags",
	DefaultFn: func() string {
		return ""
	},
	Description: "Extra flags for the xelab command",
}.Register()

// XelabDebug enables the user to control the accessibility in the compiled design for
// debugging purposes.
var XelabDebug = core.StringFlag{
	Name: "xsim-xelab-debug",
	DefaultFn: func() string {
		return "typical"
	},
	Description:   "Extra debug flags for the xelab command",
	AllowedValues: []string{"line", "wave", "drivers", "readers", "xlibs", "all", "typical", "subprogram", "off"},
}.Register()

// XsimFlags enables the user to specify additional flags for the 'xsim' command.
var XsimFlags = core.StringFlag{
	Name: "xsim-xsim-flags",
	DefaultFn: func() string {
		return ""
	},
	Description: "Extra flags for the xsim command",
}.Register()

// xsim_rules holds a map of all defined rules to prevent defining the same rule
// multiple times.
var xsim_rules map[string]bool

// Parameters of the do-file
type tclFileParams struct {
	DumpWdb     bool
	DumpVcd     bool
	DumpVcdFile string
}

// Do-file template
const xsim_tcl_file_template = `
if [info exists ::env(from)] {
	run $::env(from)
}

{{ if .DumpVcd }}
open_vcd {{ .DumpVcdFile }}
log_vcd [get_objects -r]
{{ end }}

{{ if .DumpWdb }}
log_wave [get_objects -r]
{{ end }}

if [info exists ::env(duration)] {
	run $::env(duration)
} else {
	run -all
}

{{ if .DumpVcd }}
close_vcd
{{ end }}
`

type prjFile struct {
	Rule   Simulation
	Macros []string
	Incs   []string
	Deps   []core.Path
	Data   []string
}

func addToPrjFile(ctx core.Context, prj prjFile, ips []Ip, srcs []core.Path) prjFile {
	for _, ip := range ips {
		prj = addToPrjFile(ctx, prj, ip.Ips(), ip.Sources())
	}

	for _, src := range srcs {
		if IsHeader(src.String()) {
			new_path := path.Dir(src.Absolute())
			if !xsim_rules[new_path] {
				prj.Incs = append(prj.Incs, new_path)
				xsim_rules[new_path] = true
			}
		} else if IsRtl(src.String()) {
			if xsim_rules[src.String()] {
				continue
			}

			prefix := ""
			if IsSystemVerilog(src.String()) {
				prefix = "sv"
			} else if IsVerilog(src.String()) {
				prefix = "verilog"
			} else if IsVhdl(src.String()) {
				prefix = "vhdl"
			}

			entry := fmt.Sprintf("%s %s %s", prefix, strings.ToLower(prj.Rule.Lib()), src.String())

			for _, inc_path := range prj.Incs {
				entry = entry + " -i " + inc_path
			}

			if len(prj.Macros) > 0 {
				entry = entry + " -d " + strings.Join(prj.Macros, " -d ")
			}

			prj.Data = append(prj.Data, entry)

			xsim_rules[src.String()] = true
		}

		prj.Deps = append(prj.Deps, src)
	}

	return prj
}

func createPrjFile(ctx core.Context, rule Simulation) core.Path {
	// Clear the rules map
	xsim_rules = make(map[string]bool)

	// Setup macros
	macros := []string{"SIMULATION"}
	for _, key := range sortedStringKeys(rule.Defines) {
    value := rule.Defines[key]
		macro := key
		if value != "" {
			macro = fmt.Sprintf("%s=%s", key, value)
		}
		macros = append(macros, macro)
	}
	prjFilePath := rule.Path().WithSuffix("/" + "xsim.prj")
	prjFileContents := addToPrjFile(
		ctx,
		prjFile{
			Rule:   rule,
			Macros: macros,
			Incs:   []string{core.SourcePath("").String()},
		}, rule.Ips, rule.Srcs)
	ctx.AddBuildStep(core.BuildStep{
		Out:   prjFilePath,
		Ins:   prjFileContents.Deps,
		Data:  strings.Join(prjFileContents.Data, "\n"),
		Descr: fmt.Sprintf("xsim project: %s", prjFilePath.Relative()),
	})

	return prjFilePath
}

// Create a simulation script
func tclFile(ctx core.Context, rule Simulation) {
	// Do-file script
	params := tclFileParams{
		DumpWdb:     XsimDumpWdb.Value(),
		DumpVcd:     DumpVcd.Value(),
		DumpVcdFile: rule.Path().WithSuffix(fmt.Sprintf("/%s.vcd", rule.Name)).String(),
	}

	tclFile := rule.Path().WithSuffix("/xsim.tcl")
	ctx.AddBuildStep(core.BuildStep{
		Out:   tclFile,
		Data:  core.CompileTemplate(xsim_tcl_file_template, "tcl", params),
		Descr: fmt.Sprintf("xsim: %s", tclFile.Relative()),
	})
}

// elaborate creates and optimized version of the design optionally including
// coverage recording functionality. The optimized design unit can then conveniently
// be simulated using 'xsim'.
func elaborate(ctx core.Context, rule Simulation, prj_file core.Path) {
	xelab_base_cmd := []string{
		"xelab",
		"--timescale",
		"1ns/1ps",
		"--debug", XelabDebug.Value(),
		"--prj", prj_file.String(),
		XelabFlags.Value(),
	}

	for _, lib := range rule.Libs {
		xelab_base_cmd = append(xelab_base_cmd, "--lib", lib)
	}

	tops := []string{"board"}
	if rule.Top != "" {
		tops = []string{rule.Top}
	} else if len(rule.Tops) > 0 {
		tops = rule.Tops
	}

	for _, top := range tops {
		xelab_base_cmd = append(xelab_base_cmd, strings.ToLower(rule.Lib())+"."+top)
	}

	log_file_suffix := "xelab.log"
	log_files := []core.OutPath{}
	targets := []string{}
	params := []string{}
	if rule.Params != nil {
		for key, _ := range rule.Params {
			log_files = append(log_files, rule.Path().WithSuffix("/"+key+"_"+log_file_suffix))
			targets = append(targets, rule.Name+"_"+key)
			params = append(params, key)
		}
	} else {
		log_files = append(log_files, rule.Path().WithSuffix("/"+log_file_suffix))
		targets = append(targets, rule.Name)
		params = append(params, "")
	}

	for i := range log_files {
		log_file := log_files[i]
		target := targets[i]
		param_set := params[i]

		// Build up command using base command plus additional variable arguments
		xelab_cmd := append(xelab_base_cmd, "--log", log_file.String(), "--snapshot", target)

		// Set up parameters
		if param_set != "" {
			// Check that the parameters exist
			if params, ok := rule.Params[param_set]; ok {
				// Add parameters for all generics
				for param, value := range params {
					xelab_cmd = append(xelab_cmd, "-generic_top", fmt.Sprintf("\"%s=%s\"", param, value))
				}
			} else {
				log.Fatal(fmt.Sprintf("parameter set '%s' not defined for Simulation target '%s'!",
					params, rule.Name))
			}
		}

		cmd := strings.Join(xelab_cmd, " ") + " > /dev/null || { cat " + log_file.String() +
			"; rm " + log_file.String() + "; exit 1; }"

		// Hack: Add testcase generator as an optional dependency
		deps := []core.Path{prj_file}
		if rule.TestCaseGenerator != nil {
			deps = append(deps, rule.TestCaseGenerator)
		}

		// Add the rule to run 'xelab'.
		ctx.AddBuildStep(core.BuildStep{
			Out:   log_file,
			Ins:   deps,
			Cmd:   cmd,
			Descr: fmt.Sprintf("xelab: %s %s", strings.Join(tops, " "), target),
		})
	}
}

// BuildXsim will compile and elaborate the source and IPs associated with the given
// rule.
func BuildXsim(ctx core.Context, rule Simulation) {
	prj := createPrjFile(ctx, rule)

	// compile and elaborate the code
	elaborate(ctx, rule, prj)

	// Create simulation script
	tclFile(ctx, rule)
}

// xsimVerbosityLevelToFlag takes a verbosity level of none, low, medium or high and
// converts it to the corresponding DVM_ level.
func xsimVerbosityLevelToFlag(level string) (string, bool) {
	var verbosity_flag string
	var print_output bool
	switch level {
	case "none":
		verbosity_flag = " --testplusarg verbosity=DVM_VERB_NONE"
		print_output = false
	case "low":
		verbosity_flag = " --testplusarg verbosity=DVM_VERB_LOW"
		print_output = true
	case "medium":
		verbosity_flag = " --testplusarg verbosity=DVM_VERB_MED"
		print_output = true
	case "high":
		verbosity_flag = " --testplusarg verbosity=DVM_VERB_HIGH"
		print_output = true
	case "all":
		verbosity_flag = "--testplusarg verbosity=DVM_VERB_ALL"
		print_output = true
	default:
		log.Fatal(fmt.Sprintf("invalid verbosity flag '%s', only 'low', 'medium',"+
			" 'high', 'all' or 'none' allowed!", level))
	}

	return verbosity_flag, print_output
}

// xsimCmd will create a command for starting 'xsim' on the compiled and optimized design with flags
// set in accordance with what is specified on the command line.
func xsimCmd(rule Simulation, args []string, gui bool, testcase string, params string) string {
	// Prefix the xsim command with this
	cmd_preamble := ""

	// Default log and wdb file names
	file_suffix := ""
	file_spacer := ""
	if testcase != "" {
		file_suffix = testcase
		file_spacer = "_"
	}
	if params != "" {
		if file_suffix != "" {
			file_suffix = "_" + file_suffix
		}
		file_suffix = params + file_suffix
		file_spacer = "_"
	}

	log_file := rule.Path().WithSuffix("/" + file_suffix + file_spacer + "xsim.log")
	wdb_file := rule.Path().WithSuffix("/" + file_suffix + ".wdb")

	// Script to execute
	do_file := rule.Path().WithSuffix("/" + "xsim.tcl")

	// Default flag values
	seed := int64(1)
	xsim_cmd := []string{
		"xsim",
		"--log", log_file.String(),
		"--tclbatch", do_file.String(),
		XsimFlags.Value()}
	verbosity_level := "none"

	// Parse additional arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "-seed=") {
			// Define simulator seed
			var seed_flag int64
			if _, err := fmt.Sscanf(arg, "-seed=%d", &seed_flag); err == nil {
				seed = seed_flag
			} else {
				log.Fatal("-seed expects an integer argument!")
			}
		} else if strings.HasPrefix(arg, "-from=") {
			// Define how long to run
			var from string
			if _, err := fmt.Sscanf(arg, "-from=%s", &from); err == nil {
				xsim_cmd = append([]string{fmt.Sprintf("export from=%s &&", from)}, xsim_cmd...)
			} else {
				log.Fatal("-from expects an argument of '<timesteps>[<time units>]'!")
			}
		} else if strings.HasPrefix(arg, "-duration=") {
			// Define how long to run
			var to string
			if _, err := fmt.Sscanf(arg, "-duration=%s", &to); err == nil {
				xsim_cmd = append([]string{fmt.Sprintf("export duration=%s &&", to)}, xsim_cmd...)
			} else {
				log.Fatal("-duration expects an argument of '<timesteps>[<time units>]'!")
			}
		} else if strings.HasPrefix(arg, "-verbosity=") {
			// Define verbosity level
			var level string
			if _, err := fmt.Sscanf(arg, "-verbosity=%s", &level); err == nil {
				verbosity_level = level
			} else {
				log.Fatal("-verbosity expects an argument of 'low', 'medium', 'high' or 'none'!")
			}
		} else if strings.HasPrefix(arg, "+") {
			// All '+' arguments go directly to the simulator
			xsim_cmd = append(xsim_cmd, "--testplusarg", strings.TrimPrefix(arg, "+"))
		}
	}

	// Add seed flag
	xsim_cmd = append(xsim_cmd, "--sv_seed", fmt.Sprintf("%d", seed))

	// Create optional command preamble
	cmd_preamble, testcase = Preamble(rule, testcase)

	cmd_echo := ""
	if rule.Params != nil && params != "" {
		// Update coverage database name based on parameters. We cannot merge
		// different parameter sets, do we have to make a dedicated main database
		// for this parameter set.
		cmd_echo = "Testcase " + params

		// Update with testcase if specified
		if testcase != "" {
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
			cmd_echo = "Testcase " + testcase + ":"
		} else {
			testcase = "default"
		}
	}

	// Optionally specify waveform data file
	if gui || XsimDumpWdb.Value() {
		xsim_cmd = append(xsim_cmd, "--wdb", wdb_file.String())
	}

	cmd_postamble := ""
	if gui {
		xsim_cmd = append(xsim_cmd, "--gui")
		if rule.WaveformInit != nil && strings.HasSuffix(rule.WaveformInit.String(), ".tcl") {
			xsim_cmd = append(xsim_cmd, "--tclbatch", rule.WaveformInit.String())
		}
	} else {
		xsim_cmd = append(xsim_cmd, "--onfinish quit")
		cmd_newline := ":"
		if cmd_echo != "" {
			cmd_newline = "echo"
		}

		cmd_postamble = fmt.Sprintf("|| { %s; cat %s; exit 1; }", cmd_newline, log_file.String())
	}

	// Convert verbosity flag to string and append to command
	verbosity_flag, print_output := xsimVerbosityLevelToFlag(verbosity_level)
	xsim_cmd = append(xsim_cmd, verbosity_flag)

	//Finally, add the snapshot to the command as the last element
	snapshot := rule.Name
	if params != "" {
		snapshot = snapshot + "_" + params
	}
	xsim_cmd = append(xsim_cmd, snapshot)

	// Using this part of the command we send the stdout into a black hole to
	// keep the output clean
	cmd_devnull := ""
	if !print_output {
		cmd_devnull = "> /dev/null"
	}

	cmd := fmt.Sprintf("{ echo -n %s && %s %s && "+
		"{ { ! grep -q FAIL %s; } && echo PASS; } }",
		cmd_echo, strings.Join(xsim_cmd, " "), cmd_devnull, log_file.String())
	if cmd_preamble == "" {
		cmd = cmd + " " + cmd_postamble
	} else {
		cmd = "{ { " + cmd_preamble + " } && " + cmd + " } " + cmd_postamble
	}

	// Wrap command in another layer of {} to enable chaining
	cmd = "{ " + cmd + " }"

	return cmd
}

// simulateXsim will create a command to start 'xsim' on the compiled design
// with flags set in accordance with what is specified on the command line. It will
// optionally build a chain of commands in case the rule has parameters, but
// no parameters are specified on the command line
func simulateXsim(rule Simulation, args []string, gui bool) string {
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
		for key := range rule.Params {
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
			cmd = cmd + " && " + xsimCmd(rule, args, gui, testcases[j], params[i])
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
func RunXsim(rule Simulation, args []string) string {
	return simulateXsim(rule, args, true)
}

// Test will build the design and run a simulation in batch mode.
func TestXsim(rule Simulation, args []string) string {
	return simulateXsim(rule, args, false)
}
