package xilinx

import (
	"fmt"

	"dbt-rules/RULES/core"
	"dbt-rules/RULES/hdl"
)

func init() {
	core.AssertIsBuildableTarget(&ExportSimulatorIp{})
}

type ExportScriptParams struct {
	Family    string
	Language  string
	Library   string
	Simulator string
	Output    string
}

var exportScript = `#!/usr/bin/env -S vivado -log {{ .Output }}/vivado.log -nojournal -notrace -mode batch -source
compile_simlib -directory {{ .Output }} -simulator {{ .Simulator }} -family {{ .Family }} -language {{ .Language }} -library {{ .Library }}
`

// Export the Xilinx IP blocks to the an external simulator. The target simulator selection is based on the
// `hdl-simulator` flag, currently only works for 'questa'.
type ExportSimulatorIp struct {
	// Device Family, the following choices are valid: all, kintex7, virtex7, artix7, spartan7, zynq, kintexu,
	// kintexuplus, virtexu, virtexuplus, zynquplus, zynquplusrfsoc, versal
	Family string

	// Valid choices are: verilog, vhdl, all
	Language string

	// The simulation library to compile; one of: all, unisim, simprim
	Library string
}

func (rule ExportSimulatorIp) Build(ctx core.Context) {
	if hdl.Simulator.Value() == "xsim" {
		// The simulation libraries do not have to be compiled for the xsim simulator
		// as they are part of the tool installation directory
		return
	}

	simLibs := hdl.SimulatorLibDir.Value()
	if simLibs == "" {
		simLibs = ctx.Cwd().String()
	} else if simLibs[0] != '/' {
		core.Fatal("hdl-simulator-libs needs to contain an absolute path; current value: %s", simLibs)
	}

	family := rule.Family
	if family == "" {
		family = "all"
	}

	lang := rule.Language
	if lang == "" {
		lang = "all"
	}

	lib := rule.Library
	if lib == "" {
		lib = "all"
	}

	data := ExportScriptParams{
		Family:    family,
		Language:  lang,
		Library:   lib,
		Simulator: hdl.Simulator.Value(),
		Output:    simLibs,
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:    ctx.Cwd().WithSuffix("/dummy"),
		Script: core.CompileTemplate(exportScript, "export-ip-script", data),
		Descr:  fmt.Sprintf("Exporting simulator IP for %s to %s", hdl.Simulator.Value(), simLibs),
	})
}
