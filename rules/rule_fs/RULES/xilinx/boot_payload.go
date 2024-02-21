package xilinx

import (
	"fmt"

	"dbt-rules/RULES/core"
)

func init() {
	core.AssertIsBuildableTarget(&BootPayload{})
}

type BootPayloadScriptParams struct {
	Out   core.OutPath
	Fsbl  core.Path
	PmuFw core.Path
	Bl31  core.Path
	UBoot core.Path
}

var bootPayloadScript = `#!/bin/bash
set -eu -o pipefail

TMPFILE=$(mktemp -t ci-XXXXXXXXXX)
cat > ${TMPFILE} << EOF
the_ROM_image:
{
  [bootloader, destination_cpu=a53-0]{{ .Fsbl }}
  [pmufw_image]{{ .PmuFw }}
  [destination_cpu =a53-0, exception_level =el-3, trustzone]{{ .Bl31 }}
  [destination_cpu =a53-0, exception_level =el-2]{{ .UBoot }}
}
EOF

bootgen -arch zynqmp -image ${TMPFILE} -o {{ .Out }} -w | ( grep -E "^(ERROR|WARNING|CRITICAL)" || true )

rm -f ${TMPFILE}
`

// Build the bootloader payload file for ZynqMP, combining the platform management firmware and various stages of the
// bootleader
type BootPayload struct {
	Out                core.OutPath
	Handoff            Handoff
	ArmTrustedFirmware ArmTrustedFirmware
	UBoot              UBoot
}

func (rule BootPayload) Build(ctx core.Context) {
	data := BootPayloadScriptParams{
		Out:   rule.Out,
		Fsbl:  rule.Handoff.Fsbl,
		PmuFw: rule.Handoff.PmuFw,
		Bl31:  rule.ArmTrustedFirmware.Bl31,
		UBoot: rule.UBoot.Out,
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:    rule.Out,
		Ins:    []core.Path{data.Fsbl, data.PmuFw, data.Bl31, data.UBoot},
		Script: core.CompileTemplate(bootPayloadScript, "boot-payload-script", data),
		Descr:  fmt.Sprintf("Building Boot Payload"),
	})
}
