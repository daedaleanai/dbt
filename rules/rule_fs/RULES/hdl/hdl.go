package hdl

import (
	"dbt-rules/RULES/core"
	"fmt"
	"log"
	"path"
	"strings"
)

func init() {
	core.AssertIsBuildableTarget(&Fpga{})
}

// The name of a specific FPGA evaluation board if supported by the implementation tool
var BoardName = core.StringFlag{
	Name: "board",
	DefaultFn: func() string {
		return ""
	},
}.Register()

// The name of a specific part to use for implementation
var PartName = core.StringFlag{
	Name: "part",
	DefaultFn: func() string {
		return "xczu3eg-sbva484-1-e"
	},
}.Register()

var Implementation = core.StringFlag{
	Name:        "hdl-implementation",
	Description: "Select HDL implementation tool",
	DefaultFn: func() string {
		return "vivado"
	},
	AllowedValues: []string{"vivado", "quartus"},
}.Register()

var ImplementationStep = core.StringFlag{
	Name:        "hdl-implementation-step",
	Description: "Select HDL last implementation step",
	DefaultFn: func() string {
		return "bitstream"
	},
	AllowedValues: []string{"project", "synthesis", "placement", "routing", "bitstream"},
}.Register()

var ImplementationGui = core.BoolFlag{
	Name: "hdl-implementation-gui",
	DefaultFn: func() bool {
		return false
	},
	Description: "Run implementation in a GUI",
}.Register()

type FlagMap map[string]string

type Ip interface {
	Sources() []core.Path
	Data() []core.Path
	Ips() []Ip
	Flags() FlagMap
	AllSources() []core.Path
	FilterSources(string) []core.Path
	filterSources(map[string]bool, []core.Path, string) (map[string]bool, []core.Path)
}

type Library struct {
	Srcs      []core.Path
	DataFiles []core.Path
	IpDeps    []Ip
	ToolFlags FlagMap
}

func (lib Library) Sources() []core.Path {
	return lib.Srcs
}

func (lib Library) Data() []core.Path {
	return lib.DataFiles
}

func (lib Library) Ips() []Ip {
	return lib.IpDeps
}

func (lib Library) Flags() FlagMap {
	return lib.ToolFlags
}

// Get all sources from a target, including listed IPs.
func (lib Library) AllSources() []core.Path {
	return lib.FilterSources("")
}

// Get all sources from a target that match a filter pattern, including listed IPs.
func (lib Library) FilterSources(suffix string) []core.Path {
	_, sources := lib.filterSources(map[string]bool{}, []core.Path{}, suffix)
	return sources
}

// Get sources from a target, including listed IPs recursively.
// Takes a slice of sources and a map of sources we have already seen and adds everything new from the current rule.
func (lib Library) filterSources(seen map[string]bool, sources []core.Path, suffix string) (map[string]bool, []core.Path) {
	// Add sources from dependent IPs
	for _, ipDep := range lib.Ips() {
		seen, sources = ipDep.filterSources(seen, sources, suffix)
	}

	// Add sources
	for _, source := range lib.Sources() {
		if (suffix == "") || strings.HasSuffix(source.String(), suffix) {
			if _, ok := seen[source.String()]; !ok {
				seen[source.String()] = true
				sources = append(sources, source)
			}
		}
	}

	return seen, sources
}

// Get all include directories from a target, including listed IPs.
func (lib Library) AllIncDirs() []core.Path {
	incs := []core.Path{}
	seen_incs := make(map[string]struct{})
	for _, inc := range append(lib.FilterSources(".vh"), lib.FilterSources(".svh")...) {
		inc_path := path.Dir(inc.Absolute())
		if _, ok := seen_incs[inc_path]; !ok {
			incs = append(incs, core.SourcePath(path.Dir(inc.Relative())))
			seen_incs[inc_path] = struct{}{}
		}
	}
	return incs
}

type PropMap map[string]map[string]string

type Fpga struct {
	Library
	Name      string
	Top       string
	Part      string
	Board     string
	Params    map[string]string
	Defines   map[string]string
	ToolProps PropMap
}

func (rule Fpga) Build(ctx core.Context) {
	switch Implementation.Value() {
	case "vivado":
		BuildVivado(ctx, rule)
	default:
		log.Fatal(fmt.Sprintf("invalid value '%s' for hdl-implementation flag", Implementation.Value()))
	}
}
