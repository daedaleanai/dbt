package main

import (
	"github.com/daedaleanai/dbt/cmd"

	// including all rules here so they get checked when building the module
	_ "github.com/daedaleanai/dbt/RULES/cc"
	_ "github.com/daedaleanai/dbt/RULES/core"
)

func main() {
	cmd.Execute()
}
