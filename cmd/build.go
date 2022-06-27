package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

const buildDirName = "BUILD"
const buildFileName = "BUILD.go"
const compileCommandsDbFileName = "compile_commands.json"
const compileCommandsFileName = "compile_commands.sh"
const dbtRulesDirName = "dbt-rules"
const defaultOutputDir = "OUTPUT"
const dependencyGraphFileName = "graph.dot"
const generatorDirName = "GENERATOR"
const generatorInputFileName = "input.json"
const generatorOutputFileName = "output.json"
const initFileName = "init.go"
const mainFileName = "main.go"
const modFileName = "go.mod"
const ninjaFileName = "build.ninja"
const outputDirFlagName = "output-dir"
const rulesDirName = "RULES"

const goMajorVersion = 1
const goMinorVersion = 16

const initFileTemplate = `
// This file is generated. Do not edit this file.

package %s

import "dbt-rules/RULES/core"

type __internal_pkg struct{}

func DbtMain(vars map[string]interface{}) {
%s
}

func in(name string) core.Path {
	return core.NewInPath(__internal_pkg{}, name)
}

func ins(names ...string) []core.Path {
	var paths []core.Path
	for _, name := range names {
		paths = append(paths, in(name))
	}
	return paths
}

func out(name string) core.OutPath {
	return core.NewOutPath(__internal_pkg{}, name)
}

func (ip __internal_pkg) SrcDir() string {
	return "%s"
}

`
const mainFileTemplate = `
// This file is generated. Do not edit this file.

package main

import (
	"regexp"
	"runtime"
	"strconv"

	"dbt-rules/RULES/core"
)

%s

func init() {
	requiredMajor := uint64(%d)
	requiredMinor := uint64(%d)

	re := regexp.MustCompile("^go([[:digit:]]+)\\.([[:digit:]]+)(\\.[[:digit:]]+)?$")
	matches := re.FindStringSubmatch(runtime.Version())
	if matches == nil {
		core.Fatal("Failed to determine go version")
	}
	currentMajor, _ := strconv.ParseUint(matches[1], 10, 64)
	currentMinor, _ := strconv.ParseUint(matches[2], 10, 64)

	if currentMajor < requiredMajor || (currentMajor == requiredMajor && currentMinor < requiredMinor) {
		core.Fatal("DBT requires go version >= %%d.%%d. Found %%d.%%d", requiredMajor, requiredMinor, currentMajor, currentMinor)
	}
}

func main() {
    vars := map[string]interface{}{}

%s

    core.GeneratorMain(vars)
}
`

type mode uint

const (
	modeBuild mode = 1
	modeRun   mode = 2
	modeTest  mode = 3
	modeCoverage  mode = 4
)

type target struct {
	Description string
	Runnable    bool
	Testable    bool
	Report 		bool
}

type flag struct {
	Description   string
	Type          string
	AllowedValues []string
	Value         string
}

type generatorInput struct {
	DbtVersion      [3]uint
	SourceDir       string
	WorkingDir      string
	OutputDir       string
	CmdlineFlags    map[string]string
	WorkspaceFlags  map[string]string
	CompletionsOnly bool
	RunArgs         []string
	TestArgs        []string
	Layout          string
	SelectedTargets []string

	// These fields are used by dbt-rules < v1.10.0 and must be kept for backward compatibility
	Version        uint
	BuildDirPrefix string
	BuildFlags     map[string]string
}

type generatorOutput struct {
	NinjaFile string
	Targets   map[string]target
	Flags     map[string]flag

	// This field is set by dbt-rules < v1.10.0 and must be kept for backward compatibility
	BuildDir string
}

var buildCmd = &cobra.Command{
	Use:   "build [patterns] [build flags] [--commands] [--compdb] [--graph]",
	Short: "Builds the targets",
	Long:  `Builds the targets.`,
	Run: func(cmd *cobra.Command, args []string) {
		runBuild(args, modeBuild, nil)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeBuild), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: true,
}

var (
	commandList     bool
	commandDb       bool
	dependencyGraph bool
	numThreads      int
)

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().BoolVar(&commandList, "commands", false, "Create compile commands list")
	buildCmd.Flags().BoolVar(&commandDb, "compdb", false, "Create compile commands JSON database")
	buildCmd.Flags().BoolVar(&dependencyGraph, "graph", false, "Create dependency graph")
	buildCmd.Flags().IntVarP(&numThreads, "threads", "j", -1, "Run N jobs in parallel")
}

func runBuild(args []string, mode mode, modeArgs []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	dbtRulesDir := path.Join(workspaceRoot, util.DepsDirName, dbtRulesDirName)
	if !util.DirExists(dbtRulesDir) {
		log.Fatal("You are running 'dbt build' without '%s' being available. Add that dependency, run 'dbt sync' and try again.\n", dbtRulesDirName)
		return
	}

	workspaceFlags := module.ReadModuleFile(workspaceRoot).Flags
	patterns, cmdlineFlags := parseArgs(args)
	_, legacyFlags := parseArgs(args)

	outputDir := defaultOutputDir
	if workspaceOutputDir, exists := workspaceFlags[outputDirFlagName]; exists {
		outputDir = workspaceOutputDir
		delete(workspaceFlags, outputDirFlagName)
	}
	if cmdlineOutputDir, exists := cmdlineFlags[outputDirFlagName]; exists {
		outputDir = cmdlineOutputDir
		delete(cmdlineFlags, outputDirFlagName)
	}

	if !strings.HasPrefix(outputDir, "/") {
		outputDir = path.Join(workspaceRoot, buildDirName, outputDir)
	}
	log.Debug("Output directory: %s.\n", outputDir)
	genInput := generatorInput{
		DbtVersion:     util.DbtVersion,
		OutputDir:      outputDir,
		CmdlineFlags:   cmdlineFlags,
		WorkspaceFlags: workspaceFlags,

		// Legacy fields
		Version:        2,
		BuildDirPrefix: outputDir,
		BuildFlags:     legacyFlags,
	}
	switch mode {
	case modeRun:
		genInput.RunArgs = modeArgs
	case modeTest:
		genInput.TestArgs = modeArgs
	case modeCoverage:
		genInput.TestArgs = modeArgs
	}
	genOutput := runGenerator(genInput)

	// dbt-rules < v1.10.0 will compute the build directory based on flag values and return
	// the build directory to be used by DBT.
	if genOutput.BuildDir != "" {
		genInput.OutputDir = genOutput.BuildDir
	}

	// Determine the set of targets to be built.
	log.Debug("Target patterns: '%s'.\n", strings.Join(patterns, "', '"))
	regexps := []*regexp.Regexp{}
	for _, pattern := range patterns {
		re, err := regexp.Compile(fmt.Sprintf("^%s$", pattern))
		if err != nil {
			log.Fatal("Target pattern '%s' is not a valid regular expression: %s.\n", pattern, err)
		}
		regexps = append(regexps, re)
	}
	targets := []string{}

	for name, target := range genOutput.Targets {
		if skipTarget(mode, target) {
			continue
		}

		for _, re := range regexps {
			if re.MatchString(name) {
				targets = append(targets, name)
				break
			}
		}
	}

	if mode == modeCoverage {
		// Second pass with all targets
		genInput.SelectedTargets = targets
		genOutput = runGenerator(genInput)
	}


	// Write the Ninja build file.
	ninjaFilePath := path.Join(genInput.OutputDir, ninjaFileName)
	log.Debug("Ninja file: %s.\n", ninjaFilePath)
	util.WriteFile(ninjaFilePath, []byte(genOutput.NinjaFile))

	// Print all available targets and flags if there is nothing to build.
	if !commandList && !commandDb && !dependencyGraph && len(targets) == 0 {
		targetNames := []string{}
		for name := range genOutput.Targets {
			targetNames = append(targetNames, name)
		}
		sort.Strings(targetNames)

		fmt.Println("\nAvailable targets:")
		for _, name := range targetNames {
			target := genOutput.Targets[name]
			if skipTarget(mode, target) {
				continue
			}
			fmt.Printf("  //%s", name)
			if target.Description != "" {
				fmt.Printf("  (%s)", target.Description)
			}
			fmt.Println()
		}

		// Add the output directory flag.
		genOutput.Flags[outputDirFlagName] = flag{
			Description: "Output directory",
			Type:        "string",
			Value:       outputDir,
		}

		// Sort flags alphabetically.
		flagNames := []string{}
		for name := range genOutput.Flags {
			flagNames = append(flagNames, name)
		}
		sort.Strings(flagNames)

		fmt.Println("\nAvailable flags:")
		for _, name := range flagNames {
			flag := genOutput.Flags[name]
			fmt.Printf("  %s='%s' [%s]", name, flag.Value, flag.Type)
			if len(flag.AllowedValues) > 0 {
				fmt.Printf(" ('%s')", strings.Join(flag.AllowedValues, "', '"))
			}
			if flag.Description != "" {
				fmt.Printf(" // %s", flag.Description)
			}
			fmt.Println()
		}
		return
	}

	if len(targets) > 0 {
		ninjaArgs := []string{}
		if log.Verbose {
			ninjaArgs = []string{"-v", "-d", "explain"}
		}
		if numThreads >= 0 {
			ninjaArgs = append(ninjaArgs, fmt.Sprintf("-j%d", numThreads))
		}

		suffix := ""
		switch mode {
		case modeRun:
			suffix = "#run"
		case modeTest:
			suffix = "#test"
		}

		for _, target := range targets {
			ninjaArgs = append(ninjaArgs, target+suffix)
		}
		runNinja(genInput.OutputDir, os.Stdout, ninjaArgs)
	}

	if commandList {
		args := append([]string{"-t", "commands"}, targets...)
		printNinjaOutput(genInput.OutputDir, compileCommandsFileName, "Compile commands", args)
	}
	if commandDb {
		printNinjaOutput(genInput.OutputDir, compileCommandsDbFileName, "Compile commands database", []string{"-t", "compdb"})
	}
	if dependencyGraph {
		args := append([]string{"-t", "graph"}, targets...)
		printNinjaOutput(genInput.OutputDir, dependencyGraphFileName, "Dependency graph", args)
	}
}

func runNinja(dir string, stdout io.Writer, args []string) {
	log.Debug("Running ninja command: 'ninja %s'\n", strings.Join(args, " "))
	ninjaCmd := exec.Command("ninja", args...)
	ninjaCmd.Dir = dir
	ninjaCmd.Stderr = os.Stderr
	ninjaCmd.Stdout = stdout
	err := ninjaCmd.Run()
	if err != nil {
		log.Fatal("Running ninja failed: %s\n", err)
	}
}

func printNinjaOutput(dir, fileName, label string, args []string) {
	var stdout bytes.Buffer
	runNinja(dir, &stdout, args)
	absPath := path.Join(dir, fileName)
	relPath, _ := filepath.Rel(util.GetWorkingDir(), absPath)
	util.WriteFile(absPath, stdout.Bytes())
	log.Log("\n%s: %s\n", label, relPath)

}

func completeBuildArgs(toComplete string, mode mode) []string {
	genOutput := runGenerator(generatorInput{
		DbtVersion:      util.DbtVersion,
		CompletionsOnly: true,

		// Legacy field expected by dbt-rules < v1.10.0.
		Version: 2,
	})

	if strings.Contains(toComplete, "=") {
		suggestions := []string{}
		flag := strings.SplitN(toComplete, "=", 2)[0]
		for _, value := range genOutput.Flags[flag].AllowedValues {
			suggestions = append(suggestions, fmt.Sprintf("%s=%s", flag, value))
		}
		return suggestions
	}

	suggestions := []string{}
	targetToComplete := normalizeTarget(toComplete)
	for name, target := range genOutput.Targets {
		if skipTarget(mode, target) {
			continue
		}
		if strings.Contains(name, toComplete) {
			suggestions = append(suggestions, fmt.Sprintf("//%s\t%s", name, target.Description))
		} else if strings.HasPrefix(name, targetToComplete) {
			suggestions = append(suggestions, fmt.Sprintf("%s%s\t%s", toComplete, strings.TrimPrefix(name, targetToComplete), target.Description))
		}
	}

	for name, flag := range genOutput.Flags {
		suggestions = append(suggestions, fmt.Sprintf("%s=\t%s", name, flag.Description))
	}

	return suggestions
}

func parseArgs(args []string) ([]string, map[string]string) {
	patterns := []string{}
	flags := map[string]string{}

	// Split all args into two categories: If they contain a "= they are considered
	// build flags, otherwise a target pattern to be built.
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flags[parts[0]] = parts[1]
		} else {
			patterns = append(patterns, normalizeTarget(arg))
		}
	}

	return patterns, flags
}

func normalizeTarget(target string) string {
	// Build targets are interpreted as relative to the workspace root when they start with '//'.
	// Otherwise they are interpreted as relative to the current working directory.
	// E.g.: Running 'dbt build //src/path/to/mylib.a' from anywhere in the workspace is equivalent
	// to 'dbt build mylib.a' in '.../src/path/to/' or 'dbt build path/to/mylib.a' in '.../src/'.
	if strings.HasPrefix(target, "//") {
		return strings.TrimLeft(target, "/")
	}
	endsWithSlash := strings.HasSuffix(target, "/") || target == ""
	target = path.Join(util.GetWorkingDir(), target)
	moduleRoot := util.GetModuleRootForPath(target)
	target = strings.TrimPrefix(target, path.Dir(moduleRoot))
	if endsWithSlash {
		target = target + "/"
	}
	return strings.TrimLeft(target, "/")
}

func runGenerator(input generatorInput) generatorOutput {
	workspaceRoot := util.GetWorkspaceRoot()
	input.Layout = module.ReadModuleFile(workspaceRoot).Layout
	input.SourceDir = path.Join(workspaceRoot, util.DepsDirName)
	input.WorkingDir = util.GetWorkingDir()

	// Remove all existing buildfiles.
	generatorDir := path.Join(workspaceRoot, buildDirName, generatorDirName)
	util.RemoveDir(generatorDir)

	// Copy all BUILD.go files and RULES/ files from the source directory.
	modules := module.GetAllModules(workspaceRoot)
	packages := []string{}
	for modName, module := range modules {
		modBuildfilesDir := path.Join(generatorDir, modName)
		modulePackages := copyBuildAndRuleFiles(modName, module.RootPath(), modBuildfilesDir, modules)
		packages = append(packages, modulePackages...)
	}

	createGeneratorMainFile(generatorDir, packages, modules)
	createSumGoFile(generatorDir)

	generatorInputPath := path.Join(generatorDir, generatorInputFileName)
	util.WriteJson(generatorInputPath, &input)

	cmd := exec.Command("go", "run", mainFileName)
	cmd.Dir = generatorDir
	if !input.CompletionsOnly {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
	}
	err := cmd.Run()
	if err != nil {
		log.Fatal("Failed to run generator: %s.\n", err)
	}
	var output generatorOutput
	generatorOutputPath := path.Join(generatorDir, generatorOutputFileName)
	util.ReadJson(generatorOutputPath, &output)
	return output
}

func copyBuildAndRuleFiles(moduleName, modulePath, buildFilesDir string, modules map[string]module.Module) []string {
	packages := []string{}

	log.Debug("Processing module '%s'.\n", moduleName)

	goFilesDir := path.Dir(buildFilesDir)

	for _, goMod := range module.ListGoModules(modules[moduleName]) {
		modFile := path.Join(goFilesDir, goMod.Name, modFileName)
		modFileContent := createModFileContent(goMod.Name, goMod.Deps)
		util.WriteFile(modFile, modFileContent)
	}

	buildFiles := module.ListBuildFiles(modules[moduleName])

	for _, buildFile := range buildFiles {
		relativeDirPath := strings.TrimSuffix(path.Dir(buildFile.CopyPath), "/")

		packages = append(packages, relativeDirPath)
		packageName, vars := parseBuildFile(buildFile.SourcePath)
		varLines := []string{}
		for _, varName := range vars {
			varLines = append(varLines, fmt.Sprintf("    vars[in(\"%s\").Relative()] = &%s", varName, varName))
		}

		initFileContent := fmt.Sprintf(initFileTemplate, packageName, strings.Join(varLines, "\n"), path.Dir(buildFile.SourcePath))
		initFilePath := path.Join(goFilesDir, relativeDirPath, initFileName)
		util.WriteFile(initFilePath, []byte(initFileContent))

		copyFilePath := path.Join(goFilesDir, buildFile.CopyPath)
		util.CopyFile(buildFile.SourcePath, copyFilePath)
	}

	for _, ruleFile := range module.ListRules(modules[moduleName]) {
		copyFilePath := path.Join(goFilesDir, ruleFile.CopyPath)
		util.CopyFile(ruleFile.SourcePath, copyFilePath)
	}

	return packages
}

func parseBuildFile(buildFilePath string) (string, []string) {
	fileAst, err := parser.ParseFile(token.NewFileSet(), buildFilePath, nil, parser.AllErrors)

	if err != nil {
		log.Fatal("Failed to parse '%s': %s.\n", buildFilePath, err)
	}

	vars := []string{}

	for _, decl := range fileAst.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok {
			log.Fatal("'%s' contains invalid declarations. Only import statements and 'var' declarations are allowed.\n", buildFilePath)
		}

		for _, spec := range decl.Specs {
			switch spec := spec.(type) {
			case *ast.ImportSpec:
			case *ast.ValueSpec:
				if decl.Tok.String() != "var" {
					log.Fatal("'%s' contains invalid declarations. Only import statements and 'var' declarations are allowed.\n", buildFilePath)
				}
				for _, id := range spec.Names {
					if id.Name == "_" {
						log.Warning("'%s' contains an anonymous declarations.\n", buildFilePath)
						continue
					}
					vars = append(vars, id.Name)
				}
			default:
				log.Fatal("'%s' contains invalid declarations. Only import statements and 'var' declarations are allowed.\n", buildFilePath)
			}
		}
	}

	return fileAst.Name.String(), vars
}

func createRootModFileContent(moduleName string, modules map[string]module.Module) []byte {
	mod := strings.Builder{}

	fmt.Fprintf(&mod, "module %s\n\n", moduleName)
	fmt.Fprintf(&mod, "go %d.%d\n\n", goMajorVersion, goMinorVersion)

	for _, topModule := range modules {
		for _, goModule := range module.ListGoModules(topModule) {
			fmt.Fprintf(&mod, "require %s v0.0.0\n", goModule.Name)
			fmt.Fprintf(&mod, "replace %s => ./%s\n\n", goModule.Name, goModule.Name)
		}
	}

	return []byte(mod.String())
}

func createModFileContent(moduleName string, deps []string) []byte {
	mod := strings.Builder{}

	fmt.Fprintf(&mod, "module %s\n\n", moduleName)
	fmt.Fprintf(&mod, "go %d.%d\n\n", goMajorVersion, goMinorVersion)

	for _, modName := range deps {
		fmt.Fprintf(&mod, "require %s v0.0.0\n", modName)
		fmt.Fprintf(&mod, "replace %s => ../%s\n\n", modName, modName)
	}

	return []byte(mod.String())
}

func createGeneratorMainFile(generatorDir string, packages []string, modules map[string]module.Module) {
	importLines := []string{}
	dbtMainLines := []string{}
	for idx, pkg := range packages {
		importLines = append(importLines, fmt.Sprintf("import p%d \"%s\"", idx, pkg))
		dbtMainLines = append(dbtMainLines, fmt.Sprintf("    p%d.DbtMain(vars)", idx))
	}

	mainFilePath := path.Join(generatorDir, mainFileName)
	mainFileContent := fmt.Sprintf(mainFileTemplate, strings.Join(importLines, "\n"), goMajorVersion, goMinorVersion, strings.Join(dbtMainLines, "\n"))
	util.WriteFile(mainFilePath, []byte(mainFileContent))

	modFilePath := path.Join(generatorDir, modFileName)
	modFileContent := createRootModFileContent("root", modules)
	util.WriteFile(modFilePath, modFileContent)
}

func createSumGoFile(generatorDir string) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = generatorDir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	fmt.Print(string(stderr.Bytes()))
	if err != nil {
		log.Fatal("Failed to run 'go mod download': %s.\n", err)
	}
}

func skipTarget(mode mode, target target) bool {
	switch mode {
	case modeRun:
		return !target.Runnable
	case modeTest:
		return !target.Testable
	case modeCoverage:
		return !target.Testable && !target.Report
	}
	return false
}

func sortMapKeys(m interface{}) []string {
	keys := reflect.ValueOf(m).MapKeys()
	keyList := []string{}

	for _, key := range keys {
		keyList = append(keyList, key.Interface().(string))
	}
	sort.Strings(keyList)
	return keyList
}
