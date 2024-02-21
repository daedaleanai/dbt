package rust

import (
	"dbt-rules/RULES/core"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	core.AssertIsBuildableTarget(&Binary{})
	core.AssertIsRunnableTarget(&Binary{})
}

var dockerRegexp = regexp.MustCompile(`docker-([0-9a-f]*)`)

type Binary struct {
	Out       core.OutPath
	Package   core.Path
	Toolchain string

	// If target is given, compilation is done with cross instead of cargo, as cross compilation is assumed.
	Target string
}

func (bin Binary) BinLocation() core.OutPath {
	if bin.Target != "" {
		return bin.Out.WithPrefix(fmt.Sprintf("%s/%s/release/", filepath.Base(bin.Package.Relative()), bin.Target))
	}
	return bin.Out.WithPrefix(fmt.Sprintf("%s/release/", filepath.Base(bin.Package.Relative())))
}

func (bin Binary) Build(ctx core.Context) {
	if bin.Toolchain == "" {
		bin.Toolchain = "stable"
	}

	cargo := "cargo"
	target := ""

	// Intelligently detect if we are using cross inside a container and if so make sure to override the HOSTNAME env variable so that
	// it contains the docker id.

	hostnameOverride := ""
	if bin.Target != "" {
		cargo = "cross"
		target = fmt.Sprintf("--target=%q", bin.Target)

		cgroup, err := os.ReadFile("/proc/self/cgroup")
		if err != nil {
			// Assume not running under docker
			fmt.Println("Warning: could not read /proc/self/cgroup. Assuming that we are not running under docker")
			cgroup = []byte{}
		}
		if match := dockerRegexp.FindStringSubmatch(string(cgroup)); len(match) == 2 {
			hostnameOverride = fmt.Sprint("HOSTNAME=", match[1], " ")
		}
	}

	ctx.AddBuildStep(core.BuildStep{
		Out: bin.BinLocation(),
		Ins: bin.getInputs(),
		// With suffix is used to turn an input path into an output path
		Cmd: fmt.Sprintf("cd %q && %s %s +%s build -r --locked --bins --target-dir %q %s", bin.Package, hostnameOverride, cargo, bin.Toolchain, bin.Package.WithSuffix(""), target),
	})
}

func (bin Binary) Run(args []string) string {
	quotedArgs := []string{}
	for _, arg := range args {
		quotedArgs = append(quotedArgs, fmt.Sprintf("%q", arg))
	}
	return fmt.Sprintf("%q %s", bin.BinLocation(), strings.Join(quotedArgs, " "))

}

// Depend on all files in the directory. This might be overestimate, but let `cargo` do a more precise estimate.
func (bin Binary) getInputs() []core.Path {
	inputs := []core.Path{}
	filepath.WalkDir(bin.Package.Absolute(), func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(core.SourcePath("").Absolute(), path)
		if err != nil {
			return nil
		}
		inputs = append(inputs, core.SourcePath(relPath))
		return nil
	})
	return inputs
}
