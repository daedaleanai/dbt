package golang

import (
	"bytes"
	"dbt-rules/RULES/core"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func init() {
	core.AssertIsBuildableTarget(&Binary{})
	core.AssertIsRunnableTarget(&Binary{})
}

type Binary struct {
	Out     core.OutPath
	Package core.Path
	Env     map[string]string
	Flags   []string
}

func (bin Binary) Build(ctx core.Context) {
	env := ""
	if bin.Env != nil {
		for key, value := range bin.Env {
			env = fmt.Sprintf("%s %s=%q", env, key, value)
		}
	}
	ctx.AddBuildStep(core.BuildStep{
		Out: bin.Out,
		Ins: bin.getInputs(),
		Cmd: fmt.Sprintf("cd %q && %s go build %s -o %q", bin.Package, env, strings.Join(bin.Flags, " "), bin.Out),
	})
}

func (bin Binary) Run(args []string) string {
	quotedArgs := []string{}
	for _, arg := range args {
		quotedArgs = append(quotedArgs, fmt.Sprintf("%q", arg))
	}
	return fmt.Sprintf("%q %s", bin.Out, strings.Join(quotedArgs, " "))

}

type pkg struct {
	Standard   bool
	Dir        string
	ImportPath string
	GoFiles    []string
	OtherFiles []string
	Deps       []string
	Match      []string
}

// Use 'go list' to get the source files that will be compiled into this go binary.
func (bin Binary) getInputs() []core.Path {
	cmd := exec.Command("go", "list", "-json", "-e", ".", "all")
	cmd.Dir = bin.Package.Absolute()
	data, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			core.Fatal("'go list' failed: %s\nstdout: %s\nstderr: %s\n", err, data, exitError.Stderr)
		} else {
			core.Fatal("'go list' failed: %s\nstdout: %s\n", err, data)
		}
	}

	// Create a map of all packages by import path
	pkgs := map[string]pkg{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var usedPackages []string
	for {
		var p pkg
		if err := decoder.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			core.Fatal("failed to decode 'go list' output: %s", err)
		}
		if p.Standard {
			continue
		}
		pkgs[p.ImportPath] = p
		// The current package (that we are building the binary from) is the only package that
		// matches 'all' and '.'. All files of this package and all its Deps are input files.
		if len(p.Match) > 1 {
			usedPackages = append(p.Deps, p.ImportPath)
		}
	}

	// Get all GoFiles and OtherFiles for all used packages.
	inputs := []core.Path{}
	for _, usedPackage := range usedPackages {
		p := pkgs[usedPackage]
		relPackagePath, _ := filepath.Rel(core.SourcePath("").Absolute(), p.Dir)
		for _, file := range append(p.GoFiles, p.OtherFiles...) {
			inputs = append(inputs, core.SourcePath(path.Join(relPackagePath, file)))
		}
	}
	return inputs
}
