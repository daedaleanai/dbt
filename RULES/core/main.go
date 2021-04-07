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
}

func GeneratorMain(dbtMainFns []dbtMainFn) {
	ctx := newContext()

	for _, dbtMainFn := range dbtMainFns {
		dbtMainFn(ctx.addTarget)
	}

	ctx.currentTarget = ""

	output := output{}
	output.NinjaFile = ctx.ninjaFile.String()
	output.Targets = ctx.targets

	data, err := json.MarshalIndent(output, "", "  ")
	ctx.Assert(err == nil, "failed to marshall generator output: %s", err)
	err = ioutil.WriteFile(outputFileName, data, outputFileMode)
	ctx.Assert(err == nil, "failed to write generator output: %s", err)
}
