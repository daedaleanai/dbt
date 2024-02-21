package xilinx

import (
	"fmt"
	"regexp"
	"strings"

	"dbt-rules/RULES/core"
	"dbt-rules/RULES/hdl"
)

func init() {
	core.AssertIsBuildableTarget(&Handoff{})
}

type HandoffScriptParams struct {
	HwDef      core.Path
	EmbeddedSw core.Path
	Fsbl       core.Path
	PmuFw      core.Path
	Patch      core.Path
}

var handoffScript = `#!/bin/bash
set -eu -o pipefail

TMPDIR=$(mktemp -d -t ci-XXXXXXXXXX)

(
    cd ${TMPDIR}
    cp {{ .HwDef }} design.hwdef
    cat > export.tcl << EOF
hsi::set_repo_path {{ .EmbeddedSw}}

set hw_design [hsi::open_hw_design design.hwdef]

hsi::generate_app -hw \${hw_design} -proc psu_cortexa53_0 -os standalone -app zynqmp_fsbl -dir fsbl
hsi::generate_app -hw \${hw_design} -proc psu_pmu_0 -os standalone -app zynqmp_pmufw -dir pmufw
hsi::close_hw_design [hsi::current_hw_design]
EOF
    xsct export.tcl | ( grep -E "^(ERROR|WARNING|CRITICAL)" || true )

    cd fsbl
    {{ if .Patch }}
    patch -Np1 -i {{ .Patch }}
    {{ end }}
    make > /dev/null
    cp executable.elf {{ .Fsbl }}
    cd ../pmufw
    make > /dev/null
    cp executable.elf {{ .PmuFw }}

)

rm -rf ${TMPDIR}
`

// Create the software handoff: the First Stage Bootloader and the Platform Management firmware
type Handoff struct {
	// The First Stage Bootloader
	Fsbl core.OutPath

	// The Platform Management Firmware
	PmuFw core.OutPath

	// The ZynqMP IP for which the handoff binaries are to be created
	Ip Ip

	// Platform-specific patches to be applied, if any. Go-style regexps are accepted.
	Patches []core.StringPath
}

func (rule Handoff) Build(ctx core.Context) {
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

	var patch core.Path
	board := hdl.BoardName.Value()
	for _, cfg := range rule.Patches {
		matched, err := regexp.MatchString(cfg.Key, board)
		if err != nil {
			core.Fatal("Handoff patch: %s", err)
		}
		if matched {
			patch = cfg.Value
		}
	}

	data := HandoffScriptParams{
		HwDef:      hwdef,
		EmbeddedSw: core.SourcePath("embeddedsw"),
		Fsbl:       rule.Fsbl,
		PmuFw:      rule.PmuFw,
		Patch:      patch,
	}

	outs := []core.OutPath{
		rule.Fsbl,
		rule.PmuFw,
	}

	ctx.AddBuildStep(core.BuildStep{
		Outs:   outs,
		In:     hwdef,
		Script: core.CompileTemplate(handoffScript, "handoff-script", data),
		Descr:  fmt.Sprintf("Building Handoff Software for board %s", hdl.BoardName.Value()),
	})
}
