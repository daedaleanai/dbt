package xilinx

import (
	"fmt"
	"regexp"

	"dbt-rules/RULES/core"
	"dbt-rules/RULES/hdl"
)

func init() {
	core.AssertIsBuildableTarget(&UBoot{})
}

type UBootScriptParams struct {
	Out    core.OutPath
	Repo   core.Path
	Config string
}

var uBootScript = `#!/bin/bash
set -eu -o pipefail

TMPDIR=$(mktemp -d -t ci-XXXXXXXXXX)
rsync --exclude=.git -az {{ .Repo }} ${TMPDIR}
(
    cd ${TMPDIR}/u-boot
    make ARCH=arm CROSS_COMPILE=aarch64-none-elf- {{ .Config }} > /dev/null
    make ARCH=arm CROSS_COMPILE=aarch64-none-elf- -j12 > /dev/null
    cp u-boot.elf "{{ .Out }}"
)

rm -rf ${TMPDIR}
`

// Build the U-Boot bootloader binary for the given board
type UBoot struct {
	Out core.OutPath

	// Map of board names to U-Boot configurations. Go-style regexps accepted.
	Configs []core.StringString
}

func (rule UBoot) Build(ctx core.Context) {
	var config string
	board := hdl.BoardName.Value()
	for _, cfg := range rule.Configs {
		matched, err := regexp.MatchString(cfg.Key, board)
		if err != nil {
			core.Fatal("UBoot config: %s", err)
		}
		if matched {
			config = cfg.Value
		}
	}

	if config == "" {
		core.Fatal("Unable to determine U-Boot config for board: %s", hdl.BoardName.Value())
	}

	data := UBootScriptParams{
		Out:    rule.Out,
		Repo:   core.SourcePath("u-boot"),
		Config: config,
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:    rule.Out,
		In:     core.SourcePath("u-boot"),
		Script: core.CompileTemplate(uBootScript, "uboot-script", data),
		Descr:  fmt.Sprintf("Building U-Boot for board %s", hdl.BoardName.Value()),
	})
}
