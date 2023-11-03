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
	"os/signal"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/daedaleanai/dbt/assets"
	"github.com/daedaleanai/dbt/config"
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

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
const negativeRulePrefix = "negative:"

const (
	goMajorVersion = 1
	goMinorVersion = 18
)

type mode uint

const (
	modeBuild mode = iota
	modeAnalyze
	modeCoverage
	modeList
	modeRun
	modeTest
	modeFlags
)

type target struct {
	Description string
	Runnable    bool
	Testable    bool
	Report      bool
}

type flag struct {
	Description   string
	Type          string
	AllowedValues []string
	Value         string
}

type generatorInput struct {
	DbtVersion           [3]uint
	SourceDir            string
	WorkingDir           string
	OutputDir            string
	CmdlineFlags         map[string]string
	WorkspaceFlags       map[string]string
	CompletionsOnly      bool
	RunArgs              []string
	TestArgs             []string
	Layout               string
	SelectedTargets      []string
	BuildAnalyzerTargets bool
	PersistFlags         bool

	// These fields are used by dbt-rules < v1.10.0 and must be kept for backward compatibility
	Version        uint
	BuildDirPrefix string
	BuildFlags     map[string]string
}

type generatorOutput struct {
	NinjaFile   string
	Targets     map[string]target
	Flags       map[string]flag
	CompDbRules []string

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
	buildCmd.Flags().IntVarP(&numThreads, "threads", "j", -1, "Run N jobs in parallel. Defaults to as many threads as cores available.")
}

func runBuild(args []string, mode mode, modeArgs []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	dbtRulesDir := path.Join(workspaceRoot, util.DepsDirName, dbtRulesDirName)
	if !util.DirExists(dbtRulesDir) {
		log.Fatal("You are running 'dbt build' without '%s' being available. Add that dependency, run 'dbt sync' and try again.\n", dbtRulesDirName)
		return
	}

	util.EnsureManagedDir(util.BuildDirName)

	moduleFile := module.ReadModuleFile(workspaceRoot)
	workspaceFlags := moduleFile.Flags
	positivePatterns, negativePatterns, cmdlineFlags := parseArgs(args)
	_, _, legacyFlags := parseArgs(args)

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
		outputDir = path.Join(workspaceRoot, util.BuildDirName, outputDir)
	}
	log.Debug("Output directory: %s.\n", outputDir)

	persistFlags := config.GetConfig().PersistFlags
	if moduleFile.PersistFlags != nil {
		persistFlags = *moduleFile.PersistFlags
	}
	log.Debug("Flags persistency: %t.\n", persistFlags)

	genInput := generatorInput{
		DbtVersion:           util.VersionTriplet(),
		OutputDir:            outputDir,
		CmdlineFlags:         cmdlineFlags,
		WorkspaceFlags:       workspaceFlags,
		TestArgs:             []string{},
		RunArgs:              []string{},
		BuildAnalyzerTargets: false,
		PersistFlags:         persistFlags,

		// Legacy fields
		Version:        2,
		BuildDirPrefix: outputDir,
		BuildFlags:     legacyFlags,
	}
	switch mode {
	case modeBuild:
		// do nothing
	case modeAnalyze:
		genInput.BuildAnalyzerTargets = true
	case modeCoverage:
		genInput.TestArgs = modeArgs
	case modeList, modeFlags:
		// do nothing
	case modeRun:
		genInput.RunArgs = modeArgs
	case modeTest:
		genInput.TestArgs = modeArgs
	}
	genOutput := runGenerator(genInput)

	// dbt-rules < v1.10.0 will compute the build directory based on flag values and return
	// the build directory to be used by DBT.
	if genOutput.BuildDir != "" {
		genInput.OutputDir = genOutput.BuildDir
	}

	// Determine the set of targets to be built.
	log.Debug("Target patterns: '%s'. Negative patterns: '%s'\n", strings.Join(positivePatterns, "', '"), strings.Join(negativePatterns, "', '"))
	positiveRegexps := []*regexp.Regexp{}
	for _, pattern := range positivePatterns {
		re, err := regexp.Compile(fmt.Sprintf("^%s$", pattern))
		if err != nil {
			log.Fatal("Positive target pattern '%s' is not a valid regular expression: %s.\n", pattern, err)
		}
		positiveRegexps = append(positiveRegexps, re)
	}

	negativeRegexps := []*regexp.Regexp{}
	for _, pattern := range negativePatterns {
		re, err := regexp.Compile(fmt.Sprintf("^%s$", pattern))
		if err != nil {
			log.Fatal("Negative target pattern '%s' is not a valid regular expression: %s.\n", pattern, err)
		}
		negativeRegexps = append(negativeRegexps, re)
	}

	targets := []string{}
	if mode != modeList && mode != modeFlags {
		for name, target := range genOutput.Targets {
			if skipTarget(mode, target) {
				continue
			}

			// Negative patterns have precedence
			matchesNegativePattern := false
			for _, re := range negativeRegexps {
				if re.MatchString(name) {
					matchesNegativePattern = true
					break
				}
			}

			if matchesNegativePattern {
				continue
			}

			for _, re := range positiveRegexps {
				if re.MatchString(name) {
					targets = append(targets, name)
					break
				}
			}
		}
	}

	// Second pass with all targets
	if mode == modeAnalyze || mode == modeCoverage {
		genInput.SelectedTargets = targets
		genOutput = runGenerator(genInput)
	}

	// Write the Ninja build file.
	ninjaFilePath := path.Join(genInput.OutputDir, ninjaFileName)
	log.Debug("Ninja file: %s.\n", ninjaFilePath)
	util.WriteFile(ninjaFilePath, []byte(genOutput.NinjaFile))

	// Print all available targets and flags if there is nothing to build.
	if mode == modeList {
		printTargets(genOutput, mode)
	} else if mode == modeFlags {
		printFlags(genOutput)
	} else if !commandList && !commandDb && !dependencyGraph && len(targets) == 0 {
		fmt.Println("\nAvailable targets:")
		printTargets(genOutput, mode)

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
		printFlags(genOutput)

		log.Fatal("The target is either not specified or is invalid.\n")
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
		printNinjaOutput(genInput.OutputDir,
			compileCommandsDbFileName,
			"Compile commands database",
			append([]string{"-t", "compdb"}, genOutput.CompDbRules...))
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
	err := ninjaCmd.Start()
	if err != nil {
		log.Fatal("Starting ninja failed: %s\n", err)
	}

	// Capture and handle Ctrl-C manually. Note that all subprocesses get the
	// Ctrl-C automatically nevertheless, since they belong to the same process
	// group.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGINT)

	go func() {
		<-signals
		fmt.Println("SIGINT: Waiting for ninja to finish...")

		var lastSignalTime *time.Time
		for {
			<-signals

			currentTime := time.Now()
			if lastSignalTime == nil || currentTime.Sub(*lastSignalTime) > 1*time.Second {
				fmt.Println("SIGINT: Press Ctrl-C again within 1 sec to force-kill dbt and ninja...")
				lastSignalTime = &currentTime
			} else {
				fmt.Println("SIGINT: Killing dbt, ninja and its subprocesses...")
				// Pass negative PID to kill the whole dbt process group. This
				// works only if this dbt instance is the leader of the process
				// group. Otherwise it would be unsafe to kill the whole group.
				if err := syscall.Kill(-syscall.Getpid(), syscall.SIGKILL); err != nil {
					fmt.Printf("Failed to kill dbt and ninja: %s\n", err)
				}
			}
		}
	}()

	err = ninjaCmd.Wait()
	if err != nil {
		log.Fatal("Running ninja failed: %s\n", err)
	}
	signal.Stop(signals)
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
		DbtVersion:      util.VersionTriplet(),
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
	toComplete, isNegative := util.CutPrefix(toComplete, negativeRulePrefix)
	prefix := ""
	if isNegative {
		prefix = negativeRulePrefix
	}

	targetToComplete := normalizeTarget(toComplete)
	for name, target := range genOutput.Targets {
		if skipTarget(mode, target) {
			continue
		}
		if strings.Contains(name, toComplete) {
			suggestions = append(suggestions, fmt.Sprintf("%s//%s\t%s", prefix, name, target.Description))
		} else if strings.HasPrefix(name, targetToComplete) {
			suggestions = append(suggestions, fmt.Sprintf("%s%s%s\t%s", prefix, toComplete, strings.TrimPrefix(name, targetToComplete), target.Description))
		}
	}

	for name, flag := range genOutput.Flags {
		suggestions = append(suggestions, fmt.Sprintf("%s=\t%s", name, flag.Description))
	}

	return suggestions
}

func parseArgs(args []string) ([]string, []string, map[string]string) {
	positivePatterns := []string{}
	negativePatterns := []string{}
	flags := map[string]string{}

	// Split all args into 3 categories: If they contain a "=" they are considered
	// build flags, otherwise if they start with "-" a negative target pattern
	// and otherwise a positve target pattern to be built.
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flags[parts[0]] = parts[1]
		} else {
			trimmedArg, isNegativePattern := util.CutPrefix(arg, negativeRulePrefix)

			if isNegativePattern {
				negativePatterns = append(negativePatterns, normalizeTarget(trimmedArg))
			} else {
				positivePatterns = append(positivePatterns, normalizeTarget(trimmedArg))
			}
		}
	}

	return positivePatterns, negativePatterns, flags
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
	generatorDir := path.Join(workspaceRoot, util.BuildDirName, generatorDirName)
	util.RemoveDir(generatorDir)

	// Copy all BUILD.go files and RULES/ files from the source directory.
	modules := module.GetAllModules(workspaceRoot)
	modNames := []string{}
	for modName := range modules {
		modNames = append(modNames, modName)
	}
	sort.Strings(modNames)

	packages := []string{}
	for _, modName := range modNames {
		module := modules[modName]
		modBuildfilesDir := path.Join(generatorDir, modName)
		modulePackages := copyBuildAndRuleFiles(modName, module.RootPath(), modBuildfilesDir, modules)
		packages = append(packages, modulePackages...)
	}
	sort.Strings(packages)

	createGeneratorMainFile(generatorDir, packages, modules)
	createRootModFile(path.Join(generatorDir, modFileName), modules)
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

func printTargets(genOutput generatorOutput, mode mode) {
	targetNames := []string{}
	for name := range genOutput.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

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
}

func printFlags(genOutput generatorOutput) {
	// Sort flags alphabetically.
	flagNames := []string{}
	for name := range genOutput.Flags {
		flagNames = append(flagNames, name)
	}
	sort.Strings(flagNames)

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
}

func copyBuildAndRuleFiles(moduleName, modulePath, buildFilesDir string, modules map[string]module.Module) []string {
	packages := []string{}

	log.Debug("Processing module '%s'.\n", moduleName)

	goFilesDir := path.Dir(buildFilesDir)

	for _, goMod := range module.ListGoModules(modules[moduleName]) {
		modFile := path.Join(goFilesDir, goMod.Name, modFileName)
		util.GenerateFile(modFile, *assets.Templates.Lookup(modFileName + ".tmpl"), assets.GoModTmplParams{
			RequiredGoVersionMajor: goMajorVersion,
			RequiredGoVersionMinor: goMinorVersion,
			Module:                 goMod.Name,
			Prefix:                 "../",
			Deps:                   goMod.Deps,
		})
	}

	buildFiles := module.ListBuildFiles(modules[moduleName])

	for _, buildFile := range buildFiles {
		relativeDirPath := strings.TrimSuffix(path.Dir(buildFile.CopyPath), "/")

		packages = append(packages, relativeDirPath)
		packageName, vars := parseBuildFile(buildFile.SourcePath)

		initFilePath := path.Join(goFilesDir, relativeDirPath, initFileName)
		util.GenerateFile(initFilePath, *assets.Templates.Lookup(initFileName + ".tmpl"), assets.InitFileTmplParams{
			Package:   packageName,
			Vars:      vars,
			SourceDir: path.Dir(buildFile.SourcePath),
		})

		copyFilePath := path.Join(goFilesDir, buildFile.CopyPath)
		if util.FileExists(copyFilePath) {
			log.Fatal("BUILD.go file provided by more than one dbt module: %s\n", copyFilePath)
		}
		util.CopyFile(buildFile.SourcePath, copyFilePath)
	}

	for _, ruleFile := range module.ListRules(modules[moduleName]) {
		copyFilePath := path.Join(goFilesDir, ruleFile.CopyPath)
		if util.FileExists(copyFilePath) {
			log.Fatal("Rule file provided by more than one dbt module: %s\n", copyFilePath)
		}
		util.CopyFile(ruleFile.SourcePath, copyFilePath)
	}

	sort.Strings(packages)
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
	sort.Strings(vars)

	return fileAst.Name.String(), vars
}

func createRootModFile(filePath string, modules map[string]module.Module) {
	keys := []string{}
	for key, _ := range modules {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	deps := []string{}
	for _, topModuleKey := range keys {
		topModule := modules[topModuleKey]
		for _, goModule := range module.ListGoModules(topModule) {
			deps = append(deps, goModule.Name)
		}
	}

	util.GenerateFile(filePath, *assets.Templates.Lookup(modFileName + ".tmpl"), assets.GoModTmplParams{
		RequiredGoVersionMajor: goMajorVersion,
		RequiredGoVersionMinor: goMinorVersion,
		Module:                 "root",
		Prefix:                 "./",
		Deps:                   deps,
	})
}

func createGeneratorMainFile(generatorDir string, packages []string, modules map[string]module.Module) {
	mainFilePath := path.Join(generatorDir, mainFileName)
	util.GenerateFile(mainFilePath, *assets.Templates.Lookup(mainFileName + ".tmpl"), assets.MainFileTmplParams{
		RequiredGoVersionMajor: goMajorVersion,
		RequiredGoVersionMinor: goMinorVersion,
		Packages:               packages,
	})
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
	case modeCoverage:
		return !target.Testable && !target.Report
	case modeRun:
		return !target.Runnable
	case modeTest:
		return !target.Testable
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
