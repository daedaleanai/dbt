package xilinx

import (
	"fmt"
	"regexp"
	"strings"

	"dbt-rules/RULES/core"
	"dbt-rules/RULES/hdl"
)

func init() {
	core.AssertIsBuildableTarget(&DeviceTree{})
}

type DeviceTreeScriptParams struct {
	Out            core.OutPath
	In             core.Path
	BoardDts       core.Path
	HwDef          core.Path
	DeviceTreeXlnx core.Path
}

var deviceTreeScript = `#!/bin/bash
set -eu -o pipefail

set -eu -o pipefail

TMPDIR=$(mktemp -d -t ci-XXXXXXXXXX)

(
    cd ${TMPDIR}
    cp {{ .HwDef }} design.hwdef
    cat > export.tcl << EOF
hsi::set_repo_path {{ .DeviceTreeXlnx }}

set hw_design [hsi::open_hw_design design.hwdef]
hsi::create_sw_design device-tree -os device_tree -proc psu_cortexa53_0
hsi::generate_target -dir dts
hsi::close_hw_design [hsi::current_hw_design]
EOF
    xsct export.tcl | ( grep -E "^(ERROR|WARNING|CRITICAL)" || true )
    gcc -E -nostdinc -x assembler-with-cpp -I dts {{ if ne (len .BoardDts.String) 0 }} -DBOARD_DTS="<{{ .BoardDts }}>" {{ end }} {{ .In }} -o system-assembled.dts
    dtc -O dtb -o {{ .Out }} system-assembled.dts
)

rm -rf ${TMPDIR}
`

// The system device tree description for the Linux kernel
type DeviceTree struct {
	// Final binary device tree
	Out core.OutPath

	// Top level device tree source to be compiled. I should include `system-top.dts`, the board specific source device
	// tree if any, and whatever definitions or overrides are specific for the given system.
	In core.Path

	// The ZynqMP IP block this device tree is intended for
	Ip Ip

	// A map of board specific source tree overrides. Go-style regexp are accepted, including `.*`.
	BoardDts []core.StringPath
}

func (rule DeviceTree) Build(ctx core.Context) {
	var hwdef core.Path
	for _, file := range rule.Ip.Data() {
		if strings.HasSuffix(file.String(), ".hwdef") {
			hwdef = file
			break
		}
	}

	if hwdef == nil {
		core.Fatal("Unable to find a Hardware Definition in the input design")
	}

	var boardDts core.Path
	board := hdl.BoardName.Value()
	for _, cfg := range rule.BoardDts {
		matched, err := regexp.MatchString(cfg.Key, board)
		if err != nil {
			core.Fatal("Board DTS: %s", err)
		}
		if matched {
			boardDts = cfg.Value
		}
	}

	data := DeviceTreeScriptParams{
		In:             rule.In,
		Out:            rule.Out,
		BoardDts:       boardDts,
		HwDef:          hwdef,
		DeviceTreeXlnx: core.SourcePath("device-tree-xlnx"),
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:    rule.Out,
		Ins:    []core.Path{rule.In, hwdef, boardDts},
		Script: core.CompileTemplate(deviceTreeScript, "device-tree-script", data),
		Descr:  fmt.Sprintf("Building Device Tree: %s", rule.In.Relative()),
	})
}
