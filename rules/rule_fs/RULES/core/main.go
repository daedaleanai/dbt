package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"sort"
	"unicode"
)

const inputFileName = "input.json"
const outputFileName = "output.json"

type mode uint

const (
	modeBuild mode = iota
	modeList
	modeRun
	modeTest
	modeReport
	modeFlags
)

type targetInfo struct {
	Description string
	Runnable    bool
	Testable    bool
	Report      bool
	Selected    bool
}

type generatorInput struct {
	SourceDir        string
	WorkingDir       string
	OutputDir        string
	CmdlineFlags     map[string]string
	WorkspaceFlags   map[string]string
	CompletionsOnly  bool
	RunArgs          []string
	TestArgs         []string
	Layout           string
	SelectedTargets  []string
	PersistFlags     bool
	PositivePatterns []string
	NegativePatterns []string
	Mode             mode
}

type generatorOutput struct {
	NinjaFile       string
	Targets         map[string]targetInfo
	Flags           map[string]flagInfo
	CompDbRules     []string
	SelectedTargets []string
}

var input = loadInput()

// Determine the set of targets to be built.
type targetFilter struct {
	positiveRegexps []*regexp.Regexp
	negativeRegexps []*regexp.Regexp
}

func makeFilter() targetFilter {
	filter := targetFilter{}

	for _, pattern := range input.PositivePatterns {
		re, err := regexp.Compile(fmt.Sprintf("^%s$", pattern))
		if err != nil {
			Fatal("Positive target pattern '%s' is not a valid regular expression: %s.\n", pattern, err)
		}
		filter.positiveRegexps = append(filter.positiveRegexps, re)
	}

	for _, pattern := range input.NegativePatterns {
		re, err := regexp.Compile(fmt.Sprintf("^%s$", pattern))
		if err != nil {
			Fatal("Negative target pattern '%s' is not a valid regular expression: %s.\n", pattern, err)
		}
		filter.negativeRegexps = append(filter.negativeRegexps, re)
	}

	return filter
}

func (f targetFilter) isSelected(targetPath string, info targetInfo) bool {
	// Negative patterns have precedence

	for _, re := range f.negativeRegexps {
		if re.MatchString(targetPath) {
			return false
		}
	}

	for _, re := range f.positiveRegexps {
		if re.MatchString(targetPath) {
			return true
		}
	}
	return false
}

func skipTarget(info targetInfo) bool {
	if input.Mode == modeRun && !info.Runnable {
		return true
	}

	if input.Mode == modeTest && !info.Testable {
		return true
	}

	if input.Mode != modeReport && info.Report {
		return true
	}

	return false
}

func GeneratorMain(vars map[string]interface{}) {
	output := generatorOutput{
		Targets: map[string]targetInfo{},
		Flags:   lockAndGetFlags(input.PersistFlags),
	}

	filter := makeFilter()

	var selectedTargets = []interface{}{}

	for targetPath, variable := range vars {
		targetName := path.Base(targetPath)
		if !unicode.IsUpper([]rune(targetName)[0]) {
			continue
		}
		if _, ok := variable.(BuildInterface); !ok {
			continue
		}

		info := targetInfo{}
		if descriptionIface, ok := variable.(descriptionInterface); ok {
			info.Description = descriptionIface.Description()
		}
		if _, ok := variable.(runInterface); ok {
			info.Runnable = true
		}
		if _, ok := variable.(testInterface); ok {
			info.Testable = true
		}
		if _, ok := variable.(reportInterface); ok {
			info.Report = true
		}

		if skipTarget(info) {
			continue
		}

		info.Selected = filter.isSelected(targetPath, info)

		if info.Selected {
			selectedTargets = append(selectedTargets, variable)

			if _, ok := variable.(BuildInterface); ok {
				if input.Mode != modeReport || info.Report {
					// In report mode do not pass non-report targets to ninja
					output.SelectedTargets = append(output.SelectedTargets, targetPath)
				}
			}
		}

		output.Targets[targetPath] = info
	}

	// Create build files.
	if !input.CompletionsOnly {
		ctx := newContext(vars)

		// Making sure targets are processed in a deterministic order
		targetPaths := []string{}
		for targetPath := range vars {
			targetPaths = append(targetPaths, targetPath)
		}
		sort.Strings(targetPaths)

		var allTargets = []interface{}{}
		var reportTargets = []reportInterface{}

		for _, targetPath := range targetPaths {
			tgt := vars[targetPath]
			allTargets = append(allTargets, tgt)

			if rep, ok := tgt.(reportInterface); ok {
				reportTargets = append(reportTargets, rep)
			}
		}

		for _, targetPath := range targetPaths {
			tgt := vars[targetPath]
			if rep, ok := tgt.(reportInterface); ok {
				if info, iok := output.Targets[targetPath]; !iok || !info.Selected {
					continue
				}

				tgt = rep.Report(allTargets, selectedTargets)
			}

			if build, ok := tgt.(BuildInterface); ok {
				ctx.handleTarget(targetPath, build)
			}
		}

		output.NinjaFile = ctx.ninjaFile()

		output.CompDbRules = []string{}
		for name := range ctx.compDbBuildRules {
			output.CompDbRules = append(output.CompDbRules, name)
		}
		sort.Strings(output.CompDbRules)
	}

	// Serialize generator output.
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		Fatal("failed to marshal generator output: %s", err)
	}
	err = ioutil.WriteFile(outputFileName, data, fileMode)
	if err != nil {
		Fatal("failed to write generator output: %s", err)
	}
}
