package xilinx

import (
	"fmt"

	"dbt-rules/RULES/core"
	"dbt-rules/RULES/hdl"
	h "dbt-rules/hdl"
)

func init() {
	core.AssertIsBuildableTarget(&Bitstream{})
}

var SynthFlattenStrategy = core.StringFlag{
	Name:        "xilinx-synth-flatten_strategy",
	Description: "Choose synthesis-time hierarchy flattening strategy",
	DefaultFn: func() string {
		return "rebuilt"
	},
	AllowedValues: []string{"none", "rebuilt", "full"},
}.Register()

type BuildFileScriptParams struct {
	Out             core.Path
	PartName        string
	BoardName       string
	Name            string
	IncDir          core.Path
	BoardFiles      []core.Path
	Ips             []core.Path
	Constrs         []core.Path
	Rtls            []core.Path
	OutOfContext    bool
	ReportDir       core.Path
	FlattenStrategy string
}

type RunSynthesisScriptParams struct {
	BuildScript core.Path
	Bitstream   core.Path
	DebugProbes core.Path
	Verbose     bool
	Postprocess string
}

// Build a bitstream to program the FPGA
type Bitstream struct {
	// Name of the top-level module to implement
	Name string

	// Source file defining the top-level module
	Src core.Path

	// Constraint definitions file for the design
	Constraints core.Path

	// List of IP blocks to be included
	Ips []hdl.Ip

	// Postprocessing algorithm; either "bin" (for loading with U-Boot) or ""
	Postprocess string

	// List of directories with board definitions
	BoardFiles []core.Path
	Verbose    bool
}

func (rule Bitstream) Build(ctx core.Context) {
	ips := []core.Path{}
	rtls := []core.Path{}
	constrs := []core.Path{}

	ins := []core.Path{}
	for _, ip := range hdl.FlattenIpGraph(rule.Ips) {
		for _, src := range ip.Sources() {
			if hdl.IsRtl(src.String()) {
				rtls = append(rtls, src)
			} else if hdl.IsConstraint(src.String()) {
				constrs = append(constrs, src)
			} else if hdl.IsXilinxIpCheckpoint(src.String()) {
				ips = append(ips, src)
			}
			ins = append(ins, src)

		}
	}

	outBitstream := rule.Src.WithExt("bit")
	outDebugProbes := rule.Src.WithExt("ltx")
	outBf := rule.Src.WithExt("tcl")

	// Base directory for timestamped flow reports and checkpoints (PROJECT_ROOT/synth_reports/name)
	outReportDir := core.SourcePath("../synth_reports/" + rule.Name)

	ins = append(ins, rule.Src)
	rtls = append(rtls, rule.Src)
	if rule.Constraints != nil {
		ins = append(ins, rule.Constraints)
		constrs = append(constrs, rule.Constraints)
	}

	bfData := BuildFileScriptParams{
		Out:             outBf,
		Name:            rule.Name,
		PartName:        hdl.PartName.Value(),
		BoardName:       hdl.BoardName.Value(),
		BoardFiles:      rule.BoardFiles,
		IncDir:          core.SourcePath(""),
		Ips:             ips,
		Rtls:            rtls,
		Constrs:         constrs,
		OutOfContext:    false,
		ReportDir:       outReportDir,
		FlattenStrategy: SynthFlattenStrategy.Value(),
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:    outBf,
		Ins:    ins,
		Script: core.CompileTemplateFile(h.XilinxBuildScriptTmpl.String(), bfData),
		Descr:  fmt.Sprintf("Generating a bitstream build file %s", outBf.Relative()),
	})

	rsData := RunSynthesisScriptParams{
		BuildScript: outBf,
		Bitstream:   outBitstream,
		DebugProbes: outDebugProbes,
		Verbose:     rule.Verbose,
		Postprocess: rule.Postprocess,
	}

	outs := []core.OutPath{outBitstream, outDebugProbes}
	ctx.AddBuildStep(core.BuildStep{
		Outs:   outs,
		In:     outBf,
		Script: core.CompileTemplateFile(h.XilinxRunSynthesisScriptTmpl.String(), rsData),
		Descr:  fmt.Sprintf("Generating bitstream %s", outBitstream.Relative()),
	})
}
