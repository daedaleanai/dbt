package xilinx

import (
	"dbt-rules/RULES/core"
	"dbt-rules/RULES/hdl"
	h "dbt-rules/hdl"
	"fmt"
)

func init() {
	core.AssertIsBuildableTarget(&SynthOutOfContext{})
}

type ConstraintsFileScriptParams struct {
	Out         core.Path
	ClockSignal string
	ClockPeriod float32
}

// Target for out-of-context synthesis run
type SynthOutOfContext struct {
	// Name of the top-level module to implement
	Name string

	// Top-level IP
	Ip hdl.Ip

	// Name of the clock input signal
	ClockSignal string

	// Target clock period (ns)
	ClockPeriod float32

	// Constraint definitions file for the design.
	Constraints core.Path

	// List of directories with board definitions
	BoardFiles []core.Path
}

func (rule SynthOutOfContext) Build(ctx core.Context) {
	ips := []core.Path{}
	rtls := []core.Path{}
	constrs := []core.Path{}

	ins := []core.Path{}
	for _, ip := range hdl.FlattenIpGraph([]hdl.Ip{rule.Ip}) {
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

	// Default parameters
	clockSignal := "clk_i"
	clockPeriod := float32(1.550)

	if rule.ClockSignal != "" {
		clockSignal = rule.ClockSignal
	}

	if rule.ClockPeriod != 0.0 {
		clockPeriod = rule.ClockPeriod
	}

	if rule.Constraints != nil {
		ins = append(ins, rule.Constraints)
		constrs = append(constrs, rule.Constraints)
	} else {
		// Use default out-of-context constraints - Meant for module analysis, NOT hierarchical design.
		outConstr := ctx.Cwd().WithSuffix("/" + rule.Name + "_constraints.xdc")

		cfData := ConstraintsFileScriptParams{
			Out:         outConstr,
			ClockSignal: clockSignal,
			ClockPeriod: clockPeriod,
		}

		ctx.AddBuildStep(core.BuildStep{
			Out:   outConstr,
			Data:  core.CompileTemplateFile(h.XilinxOutOfContextConstraintsTmpl.String(), cfData),
			Descr: fmt.Sprintf("Generating automatic out-of-context constraints file: %s.", outConstr.Relative()),
		})

		ins = append(ins, outConstr)
		constrs = append(constrs, outConstr)
	}

	outBf := ctx.Cwd().WithSuffix("/" + rule.Name + "_synth.tcl")

	// Base directory for timestamped flow reports and checkpoints (PROJECT_ROOT/synth_reports/name)
	outReportDir := core.SourcePath("../synth_reports/" + rule.Name)

	bfData := BuildFileScriptParams{
		Out:             outBf,
		Name:            rule.Name,
		OutOfContext:    true,
		PartName:        hdl.PartName.Value(),
		BoardName:       hdl.BoardName.Value(),
		BoardFiles:      rule.BoardFiles,
		IncDir:          core.SourcePath(""),
		Ips:             ips,
		Rtls:            rtls,
		Constrs:         constrs,
		ReportDir:       outReportDir,
		FlattenStrategy: SynthFlattenStrategy.Value(),
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:    outBf,
		Ins:    ins,
		Script: core.CompileTemplateFile(h.XilinxBuildScriptTmpl.String(), bfData),
		Descr:  fmt.Sprintf("Generating synthesis script: %s", outBf.Relative()),
	})
}
