package core

import (
	"encoding/json"
	"io/ioutil"
)

const outputFileMode = 0755
const outputFileName = "output.json"

type registerTargetFn = func(OutPath, string, interface{})
type dbtMainFn = func(registerTargetFn)

type output struct {
	NinjaFile string
	Targets   map[string]string
	Flags     map[string]string
}

func GeneratorMain(dbtMainFns []dbtMainFn) {
	ctx := newContext(false)

	for _, dbtMainFn := range dbtMainFns {
		dbtMainFn(ctx.addTarget)
	}

	currentTarget = ""

	output := output{}
	output.NinjaFile = ctx.ninjaFile.String()
	output.Targets = ctx.targets
	output.Flags = BuildFlags

	data, err := json.MarshalIndent(output, "", "  ")
	Assert(err == nil, "failed to marshall generator output: %s", err)
	err = ioutil.WriteFile(outputFileName, data, outputFileMode)
	Assert(err == nil, "failed to write generator output: %s", err)
}
