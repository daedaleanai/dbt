package hdl

import (
	"dbt-rules/RULES/core"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

type XciValue struct {
	Value string `json:"value"`
}

type XciProjectParameters struct {
	Architecture        []XciValue `json:"ARCHITECTURE"`
	BaseBoardPart       []XciValue `json:"BASE_BOARD_PART"`
	BoardConnections    []XciValue `json:"BOARD_CONNECTIONS"`
	Device              []XciValue `json:"DEVICE"`
	Package             []XciValue `json:"PACKAGE"`
	Prefhdl             []XciValue `json:"PREFHDL"`
	SiliconRevision     []XciValue `json:"SILICON_REVISION"`
	SimulatorLanguage   []XciValue `json:"SIMULATOR_LANGUAGE"`
	Speedgrade          []XciValue `json:"SPEEDGRADE"`
	StaticPower         []XciValue `json:"STATIC_POWER"`
	TemperatureGrade    []XciValue `json:"TEMPERATURE_GRADE"`
	UseRdiCustomization []XciValue `json:"USE_RDI_CUSTOMIZATION"`
	UseRdiGeneration    []XciValue `json:"USE_RDI_GENERATION"`
}

type XciParameters struct {
	ComponentParameters map[string]interface{} `json:"component_parameters"`
	ModelParameters     map[string]interface{} `json:"model_parameters"`
	ProjectParameters   XciProjectParameters   `json:"project_parameters"`
	RuntimeParameters   map[string]interface{} `json:"runtime_parameters"`
}

type XciIpInst struct {
	XciName            string                 `json:"xci_name"`
	ComponentReference string                 `json:"component_reference"`
	IpRevision         string                 `json:"ip_revision"`
	GenDirectory       string                 `json:"gen_directory"`
	Parameters         XciParameters          `json:"parameters"`
	Boundary           map[string]interface{} `json:"boundary"`
}

type Xci struct {
	Schema string    `json:"schema"`
	IpInst XciIpInst `json:"ip_inst"`
}

func ReadXci(path string) (Xci, error) {
	var result Xci

	xci_file, err := os.Open(path)
	if err == nil {
		// defer the closing of the file
		defer xci_file.Close()

		bytes, _ := ioutil.ReadAll(xci_file)

		err = json.Unmarshal([]byte(bytes), &result)
	}

	return result, err
}

type exportTemplateParams struct {
	Sources   []core.Path
	Simulator string
	Name      string
	Args      []string
	Part      string
	Board     string
	Dir       string
	LibDir    string
	Defines   []string
	Options   []string
}

const export_ip_template = `
{{- range .Sources }}
{{- if or (hasSuffix .String ".xci") }}
if {[file exists .srcs] && [file isdirectory .srcs]} {
  set name [file tail {{ .String }}]
  foreach xci [exec find .srcs -name "*.xci"] {
    if {[file tail $xci] == $name} {
      puts "Removing existing IP in $xci"
      file delete -force $xci
      break
    }
  }
}
puts "Reading IP from {{ .String }}"
import_ip {{ .String }}
{{- end }}
{{- end }}
foreach ip [get_ips] {
  puts "Upgrade IP"
  upgrade_ip $ip
  puts "Generating IP"
  generate_target simulation $ip
  puts "Exporting IP to {{ .Dir }}"
  export_simulation -simulator {{ .Simulator }} -quiet -force -absolute_path -use_ip_compiled_libs -lib_map_path {{ .LibDir }} -of_objects $ip -step compile -directory {{ .Dir }}
}
`

const vivado_command = `#!/usr/bin/env -S vivado -nojournal -nolog -mode batch -source`

const create_project_template = `
{{- if .Dir }}
if [file exists {{ .Dir }}] {
  file delete -force -- {{ .Dir }}
}
{{- end }}
create_project -in_memory -part {{ .Part }}
set_property target_language verilog [current_project]
set_property source_mgmt_mode All [current_project]
{{- if .Board }}
catch {set_property board_part {{ .Board }} [current_project]}
{{- end }}
`

const export_add_files_template = `
{{- /* add HDL source files */}}
catch {
  add_files -norecurse {
{{- range .Sources }}
{{- if or (hasSuffix .String ".v") }}
    {{ . }}
{{- end }}
{{- end }}
  }
}

{{- /* add utilities fileset */}}
if {[string equal [get_filesets -quiet utils_1] ""]} {
  create_fileset -constrset utils_1
}
catch {
  add_files -fileset utils_1 {
{{- range .Sources }}
{{- if hasSuffix .String ".tcl"}}
    {{ . }}
  {{- end }}
{{- end }}
  }
}

update_compile_order -fileset sources_1
`

const export_simulation_template = `
  set_property top {{ .Name }} [current_fileset -simset]
  export_simulation -simulator {{ .Simulator }}\
    -force -absolute_path\
    -use_ip_compiled_libs\
    -lib_map_path {{ .LibDir }}\
    -step compile\
{{- if .Defines }}
    -define [list\
{{- range .Defines }}
      { {{- . -}} }\
{{- end }}
    ]\
{{- end }}
{{- if .Options }}
    -more_options [list\
{{- range .Options }}
      { {{- $.Simulator }}.compile.{{ . -}} }\
{{- end }}
    ]\
{{- end }}
    -directory {{ .Dir }}\
`

const source_utils = `
foreach f [get_files -of [get_filesets utils_1]] {
  if {[string match *_pre_*.tcl $f] || [string match *_post_*.tcl $f]} {
    continue
  } else {
    puts "INFO: Sourcing utility file $f"
    source $f
  }
}
`

func ExportXilinxIpCheckpoint(ctx core.Context, rule Simulation, src core.Path, def DefineMap, flags FlagMap) core.Path {
	xci, err := ReadXci(src.String())
	if err != nil {
		log.Fatal(fmt.Sprintf("unable to read XCI file %s", src.Relative()))
	}

	if SimulatorLibDir.Value() == "" {
		log.Fatal("hdl-simulator-lib-dir must be set when compiling XCI files!")
	}

	part := xci.IpInst.Parameters.ProjectParameters.Device[0].Value + "-" +
		xci.IpInst.Parameters.ProjectParameters.Package[0].Value +
		xci.IpInst.Parameters.ProjectParameters.Speedgrade[0].Value

	if xci.IpInst.Parameters.ProjectParameters.TemperatureGrade[0].Value != "" {
		part = part + "-" + xci.IpInst.Parameters.ProjectParameters.TemperatureGrade[0].Value
	}

	defines := []string{"SIMULATION"}
	for key, value := range def {
		if value != "" {
			defines = append(defines, fmt.Sprintf("%s=%s", key, value))
		} else {
			defines = append(defines, key)
		}
	}

	options := []string{}
	for tool, option := range flags {
		options = append(options, fmt.Sprintf("%s:%s", tool, option))
	}

	// Determine name of .do file
	oldExt := path.Ext(src.Relative())
	newRel := strings.TrimSuffix(src.Relative(), oldExt)
	dir := core.BuildPath(path.Dir(src.Relative()))
	do := core.BuildPath(newRel).WithSuffix(fmt.Sprintf("/%s/compile.do", Simulator.Value()))

	// Template parameters are the direct and parent script sources.
	data := exportTemplateParams{
		Sources:   []core.Path{src},
		Dir:       dir.Absolute(),
		Part:      strings.ToLower(part),
		Simulator: Simulator.Value(),
		LibDir:    SimulatorLibDir.Value(),
		Defines:   defines,
		Options:   options,
	}

	ctx.AddBuildStep(core.BuildStep{
		Out:    do,
		In:     src,
		Script: core.CompileTemplate(vivado_command+create_project_template+export_ip_template, "export_ip", data),
		Descr:  fmt.Sprintf("export: %s", src.Relative()),
	})

	return do
}

type BlockDesign struct {
	Library
	Name string
	Args []string
}

func ExportBlockDesign(ctx core.Context, rule BlockDesign, def DefineMap, flags FlagMap) core.Path {
	// Get all Verilog sources files
	sources := rule.FilterSources(".tcl")
	sources = append(sources, rule.FilterSources(".v")...)

	// Select a suitable part
	part := PartName.Value()
	if val, ok := flags["part"]; ok {
		part = val
	}

	board := BoardName.Value()
	if val, ok := flags["board"]; ok {
		board = val
	}

	defines := []string{"SIMULATION"}
	for _, key := range sortedStringKeys(def) {
		// Iterate over the defines map in sorted order to make sure we don't change the script
		// unnecessarily and add a suitable command for each define
		value := def[key]
		if value != "" {
			defines = append(defines, fmt.Sprintf("%s=%s", key, value))
		} else {
			defines = append(defines, key)
		}
	}

	options := []string{}
	for _, tool := range sortedStringKeys(flags) {
		// Iterate over the tool flags in sorted fashion and create a suitable
		// command accordingly
		option := flags[tool]
		options = append(options, fmt.Sprintf("%s:%s", tool, option))
	}

	// Template parameters are the direct and parent script sources.
	data := exportTemplateParams{
		Sources:   sources,
		Dir:       ctx.Cwd().Absolute(),
		Name:      rule.Name,
		Part:      strings.ToLower(part),
		Board:     strings.ToLower(board),
		Simulator: Simulator.Value(),
		LibDir:    SimulatorLibDir.Value(),
		Defines:   defines,
		Options:   options,
	}

	do := ctx.Cwd().WithSuffix(fmt.Sprintf("/%s/compile.do", Simulator.Value()))

	ctx.AddBuildStep(core.BuildStep{
		Ins: sources,
		Out: do,
		Script: core.CompileTemplate(
			vivado_command+
				create_project_template+
				export_add_files_template+
				source_utils+
				rule.Name+
				" "+strings.Join(rule.Args, " ")+
				export_simulation_template, "export_bd", data),
		Descr: fmt.Sprintf("export: %s", rule.Name),
	})

	return do
}

type implementationTemplateParams struct {
	Top          string
	Gui          bool
	Sources      []core.Path
	IncDirs      []core.Path
	BlockDesigns []BlockDesign
	Step         string
	Part         string
	Board        string
	Dir          core.Path
	Params       map[string]string
	Defines      map[string]string
	Properties   map[string]map[string]string
}

const implementation_project_template = `#!/usr/bin/env -S vivado -mode {{ if .Gui }}gui{{ else }}batch{{ end }} -nojournal -log {{ .Dir.String }}/{{ .Top }}.log -source
create_project -force -part {{ .Part }} {{ .Top }} {{ .Dir.String }}
set_property target_language verilog [current_project]
set_property source_mgmt_mode All [current_project]
{{- if .Board }}
catch {
  set_property board_part {{ .Board }} [current_project]
  reset_property board_connections [current_project]
}
{{- end }}

# Create 'sources_1' fileset (if not found)
if {[string equal [get_filesets -quiet sources_1] ""]} {
  create_fileset -srcset sources_1
}

# Configure include directories
set_property include_dirs {\
{{- range .IncDirs }}
  {{ . }} \
{{- end }}
} [get_filesets sources_1]

# and Verilog defines
set_property verilog_define {\
{{- range $key, $value := .Defines }}
  "{{ $key }}{{ if $value }}={{ $value }}{{ end }}"\
{{- end }}
} [get_filesets sources_1]

# and parameters
set_property generic {\
{{- range $key, $value := .Params }}
  "{{ $key }}={{ $value }}"\
{{- end }}
} [get_filesets sources_1]

# Add all HDL source files
add_files -norecurse {
{{- range .Sources }}
  {{- if or (hasSuffix .String ".vhd") (hasSuffix .String ".vhdl") (hasSuffix .String ".v") (hasSuffix .String ".vh") (hasSuffix .String ".sv") (hasSuffix .String ".svh") }}
  {{ . }}
  {{- end }}
{{- end }}
}

# Add all IP source files
catch {
  import_ip {
{{- range .Sources }}
  {{- if hasSuffix .String ".xci"}}
    {{ . }}
  {{- end }}
{{- end }}
  }
}

# Add data files for simulation
if {[string equal [get_filesets -quiet sim_1] ""]} {
  create_fileset -simset sim_1
}
catch {
  add_files -fileset sim_1 {
{{- range .Sources }}
  {{- if or (hasSuffix .String ".dat") (hasSuffix .String ".hex") }}
    {{ . }}
  {{- end }}
{{- end }}
  }
}

# Add constraint files
if {[string equal [get_filesets -quiet constrs_1] ""]} {
  create_fileset -constrset constrs_1
}
catch {
  add_files -fileset constrs_1 {
{{- range .Sources }}
  {{- if hasSuffix .String ".xdc"}}
    {{ . }}
  {{- end }}
{{- end }}
  }
}

# Add utilities
if {[string equal [get_filesets -quiet utils_1] ""]} {
  create_fileset -constrset utils_1
}
catch {
  add_files -fileset utils_1 {
{{- range .Sources }}
  {{- if hasSuffix .String ".tcl"}}
    {{ . }}
  {{- end }}
{{- end }}
  }
}

# Source all utility files
foreach f [get_files -of [get_filesets utils_1]] {
  if {[string match *_pre_*.tcl $f] || [string match *_post_*.tcl $f]} {
    continue
  } else {
    puts "INFO: Sourcing utility file $f"
    source $f
  }
}

# Create block designs
{{- range .BlockDesigns }}
{{ .Name }}{{ range .Args }} {{ . }}{{ end }}
{{- end }}

set_property top {{ .Top }} [current_fileset]
if {[get_property top [current_fileset]] != "{{ .Top }}"} {
  puts "ERROR: Unable to set top to {{ .Top }}!"
  exit 1
}
puts "INFO: Project [current_project] created"

# Unlock IPs
foreach ip [get_ips *] {
  puts "INFO: Upgrading IP ${ip}"
  upgrade_ip $ip
}

# Properties
{{- range $run, $props := .Properties }}
  {{- range $name, $value := $props }}
set_property {{ $name }} "{{ $value }}" [get_runs {{ $run }}]
  {{- end }}
{{- end }}

# Configure scripts
foreach f [get_files -of [get_filesets utils_1]] {
  if [regexp -nocase {.*_(pre|post)_(\w+).tcl} $f total pre_or_post step] {
    if {$step == "synthesis"} {
      set property "synth_design"
      set run "synth_*"
    } else {
      set property $step
      set run "impl_*"
    }

    puts "INFO: Configuring ${pre_or_post}-$step script $f"
    set_property steps.$property.tcl.$pre_or_post $f [get_runs $run]
  }
}

{{- if eq .Step "project" }}
{{- if .Gui }}
break
{{- else }}
exit 0
{{- end }}
{{- end }}

puts "INFO: Running synthesis"
reset_run synth_1
launch_runs synth_1
wait_on_run synth_1
puts "INFO: Synthesis done"

puts "INFO: Timing summary after synthesis"
open_run synth_1
report_timing_summary

{{- if eq .Step "synthesis" }}
{{- if .Gui }}
break
{{- else }}
exit 0
{{- end }}
{{- end }}

puts "INFO: Running implementation"
launch_runs impl_1 -to_step {{ .Step }}
wait_on_run impl_1
puts "INFO: Implementation done"

puts "INFO: Timing summary after implementation to {{ .Step }}"
open_run impl_1
report_timing_summary
`

// Get all BlockDesigns of an Ip
func allBds(ip Ip) []BlockDesign {
	bds := []BlockDesign{}

	if bd, ok := ip.(BlockDesign); ok {
		bds = append(bds, bd)
	}

	for _, ipDep := range ip.Ips() {
		bds = append(bds, allBds(ipDep)...)
	}

	return bds
}

func BuildVivado(ctx core.Context, rule Fpga) {
	sources := rule.AllSources()
	dir := core.BuildPath("/" + rule.Top)
	project := dir.WithSuffix("/" + rule.Top + ".xpr")
	if rule.Name != "" {
		dir = core.BuildPath("/" + rule.Name)
		project = dir.WithSuffix("/" + rule.Name + ".xpr")
	}
	step := ImplementationStep.Value()
	switch step {
	case "placement":
		step = "place_design"
	case "routing":
		step = "route_design"
	case "bitstream":
		step = "write_bitstream"
	}

	data := implementationTemplateParams{
		Top:          rule.Top,
		Gui:          ImplementationGui.Value(),
		Sources:      sources,
		IncDirs:      rule.AllIncDirs(),
		BlockDesigns: allBds(rule),
		Part:         rule.Part,
		Board:        rule.Board,
		Dir:          dir,
		Params:       rule.Params,
		Defines:      rule.Defines,
		Step:         step,
	}

	ctx.AddBuildStep(core.BuildStep{
		Ins:    sources,
		Out:    project,
		Script: core.CompileTemplate(implementation_project_template, "implementation", data),
		Descr:  fmt.Sprintf("vivado: %s", rule.Top),
	})
}
