package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/crc32"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

const buildDirName = "BUILD"
const buildFileName = "BUILD.go"
const generatorDirName = "GENERATOR"
const generatorOutputFileName = "output.json"
const initFileName = "init.go"
const mainFileName = "main.go"
const modFileName = "go.mod"
const ninjaFileName = "build.ninja"
const outputDirPrefix = "OUTPUT"
const rulesDirName = "RULES"

const goVersion = "1.13"

const initFileTemplate = `
// This file is generated. Do not edit this file.

package %s

import "dbt-rules/RULES/core"

type __internal_pkg struct{}

func DbtGetVariables() map[string]interface{} {
    return map[string]interface{}{
%s
	}
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
`

const mainFileTemplate = `
// This file is generated. Do not edit this file.

package main

import "dbt-rules/RULES/core"

%s

func merge(dst map[string]interface{}, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

func main() {
    vars := map[string]interface{}{}

%s

    core.GeneratorMain(vars)
}
`

type target struct {
	Description string
}

type flag struct {
	Type string
	Alias string
	AllowedValues []string
	Value string
}

type generatorOutput struct {
	NinjaFile string
	Targets   map[string]target
	Flags     map[string]flag
	outputDir string
}

var buildCmd = &cobra.Command{
	Use:                   "build [targets] [build flags]",
	Short:                 "Builds the targets",
	Long:                  `Builds the targets.`,
	Run:                   runBuild,
	ValidArgsFunction:     completeArgs,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) {
	targets, flags := parseArgs(args)
	genOutput := runGenerator("ninja", flags, false)

	log.Debug("Targets: '%s'.\n", strings.Join(targets, "', '"))

	// Get all available targets and flags.
	if len(targets) == 0 {
		log.Debug("No targets specified.\n")

		fmt.Println("\nAvailable targets:")
		for name, target := range genOutput.Targets {
			fmt.Printf("  //%s", name)
			if target.Description != "" {
				fmt.Printf("  //%s (%s)", name, target.Description)
			}
			fmt.Println()	
		}

		fmt.Println("\nAvailable flags:")
		for name, flag := range genOutput.Flags {
			fmt.Printf("  %s='%s' [%s]", name, flag.Value, flag.Type)
			if len(flag.AllowedValues) > 0{
				fmt.Printf(" ('%s')", strings.Join(flag.AllowedValues, "', '"))
			}
			fmt.Println()
		}
		return
	}

	expandedTargets := map[string]struct{}{}
	for _, target := range targets {
		if !strings.HasSuffix(target, "...") {
			if _, exists := genOutput.Targets[target]; !exists {
				log.Fatal("Target '%s' does not exist.\n", target)
			}
			expandedTargets[target] = struct{}{}
			continue
		}

		targetPrefix := strings.TrimSuffix(target, "...")
		found := false
		for availableTarget := range genOutput.Targets {
			if strings.HasPrefix(availableTarget, targetPrefix) {
				found = true
				expandedTargets[availableTarget] = struct{}{}
			}
		}
		if !found {
			log.Fatal("No target is matching pattern '%s'.\n", target)
		}
	}

	// Write the ninja.build file.
	ninjaFilePath := path.Join(genOutput.outputDir, ninjaFileName)
	util.WriteFile(ninjaFilePath, []byte(genOutput.NinjaFile))

	// Run ninja.
	ninjaArgs := []string{}
	if log.Verbose {
		ninjaArgs = append(ninjaArgs, "-v")
	}
	for target := range expandedTargets {
		ninjaArgs = append(ninjaArgs, target)
	}
	
	log.Debug("Running ninja command: 'ninja %s'\n", strings.Join(ninjaArgs, " "))
	ninjaCmd := exec.Command("ninja", ninjaArgs...)
	ninjaCmd.Dir = genOutput.outputDir
	ninjaCmd.Stderr = os.Stderr
	ninjaCmd.Stdout = os.Stdout
	err := ninjaCmd.Run()
	if err != nil {
		log.Fatal("Running ninja failed: %s\n", err)
	}
}

func completeArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	//_, flags := parseArgs()
	//genOutput := runGenerator(args, "completion")

	suggestions := []string{}
/*	for flag := range getAvailableFlags(info) {
		suggestions = append(suggestions, fmt.Sprintf("%s=", flag))
	}

	targetToComplete := normalizeTarget(toComplete)
	numParts := len(strings.Split(targetToComplete, "/"))
	for target := range getAvailableTargets(info) {
		if !strings.HasPrefix(target, targetToComplete) {
			continue
		}
		suggestion := strings.Join(strings.SplitAfter(target, "/")[0:numParts], "")
		suggestion = toComplete + strings.TrimPrefix(suggestion, targetToComplete)
		suggestions = append(suggestions, suggestion)
	}

	sort.Strings(suggestions)
*/
	return suggestions, cobra.ShellCompDirectiveNoSpace
}

func parseArgs(args []string) ([]string, []string) {
	targets := []string{}
	flags := []string{}

	// Split all args into two categories: If they contain a "= they are considered
	// build flags, otherwise a target to be built.
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			flags = append(flags, arg)
		} else {
			targets = append(targets, normalizeTarget(arg))
		}
	}

	return targets, flags
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

func runGenerator(mode string, flags []string, silent bool) generatorOutput {
	workspaceRoot := util.GetWorkspaceRoot()
	sourceDir := path.Join(workspaceRoot, util.DepsDirName)
	workingDir := util.GetWorkingDir()


	// Create a hash from all sorted build flags and a unique output directory for this set of flags.
	sort.Strings(flags)
	buildConfigHash := crc32.ChecksumIEEE([]byte(strings.Join(flags, "#")))
	outputDirName := fmt.Sprintf("%s-%08X", outputDirPrefix, buildConfigHash)
	outputDir := path.Join(workspaceRoot, buildDirName, outputDirName)
	generatorDir := path.Join(workspaceRoot, buildDirName, generatorDirName)

	log.Debug("Source directory: '%s'.\n", sourceDir)
	log.Debug("Output directory: '%s'.\n", outputDir)

	// Remove all existing buildfiles.
	util.RemoveDir(generatorDir)

	// Copy all BUILD.go files and RULES/ files from the source directory.
	modules := module.GetAllModulePaths(workspaceRoot)
	packages := []string{}
	for modName, modPath := range modules {
		modBuildfilesDir := path.Join(generatorDir, modName)
		modulePackages := copyBuildAndRuleFiles(modName, modPath, modBuildfilesDir, modules)
		packages = append(packages, modulePackages...)
	}

	createGeneratorMainFile(generatorDir, packages, modules)
	createSumGoFile(generatorDir)

	cmdArgs := append([]string{"run", mainFileName, mode, sourceDir, outputDir, workingDir}, flags...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = generatorDir
	if !silent {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
	}
	err := cmd.Run()
	if err != nil {
		log.Fatal("Failed to run generator: %s.\n", err)
	}
	generatorOutputPath := path.Join(generatorDir, generatorOutputFileName)
	outputBytes, err := ioutil.ReadFile(generatorOutputPath)
	if err != nil {
		log.Fatal("Failed to read generator output: %s.\n", err)
	}

	var output generatorOutput
	output.outputDir = outputDir
	err = json.Unmarshal(outputBytes, &output)
	if err != nil {
		log.Fatal("Failed to parse generator output: %s.\n", err)
	}

	return output
}

func copyBuildAndRuleFiles(moduleName, modulePath, buildFilesDir string, modules map[string]string) []string {
	packages := []string{}

	log.Debug("Processing module '%s'.\n", moduleName)

	modFileContent := createModFileContent(moduleName, modules, "..")
	util.WriteFile(path.Join(buildFilesDir, modFileName), modFileContent)

	buildFiles := []string{}
	err := util.WalkSymlink(modulePath, func(filePath string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativeFilePath := strings.TrimPrefix(filePath, modulePath+"/")

		// Ignore the BUILD/, DEPS/ and RULES/ directories.
		if file.IsDir() && (relativeFilePath == buildDirName || relativeFilePath == util.DepsDirName || relativeFilePath == rulesDirName) {
			return filepath.SkipDir
		}

		// Skip everything that is not a BUILD.go file.
		if file.IsDir() || file.Name() != buildFileName {
			return nil
		}

		log.Debug("Found build file '%s'.\n", path.Join(modulePath, relativeFilePath))
		buildFiles = append(buildFiles, filePath)
		return nil
	})

	if err != nil {
		log.Fatal("Failed to search module '%s' for '%s' files: %s.\n", moduleName, buildFileName, err)
	}

	for _, buildFile := range buildFiles {
		relativeFilePath := strings.TrimPrefix(buildFile, modulePath+"/")
		relativeDirPath := strings.TrimSuffix(path.Dir(relativeFilePath), "/")

		packages = append(packages, path.Join(moduleName, relativeDirPath))
		packageName, vars := parseBuildFile(buildFile)
		varLines := []string{}
		for _, varName := range vars {
			varLines = append(varLines, fmt.Sprintf("        in(\"%s\").Relative(): &%s,", varName, varName))
		}

		initFileContent := fmt.Sprintf(initFileTemplate, packageName, strings.Join(varLines, "\n"))
		initFilePath := path.Join(buildFilesDir, relativeDirPath, initFileName)
		util.WriteFile(initFilePath, []byte(initFileContent))

		copyFilePath := path.Join(buildFilesDir, relativeFilePath)
		util.CopyFile(buildFile, copyFilePath)
	}

	rulesDirPath := path.Join(modulePath, rulesDirName)
	if !util.DirExists(rulesDirPath) {
		log.Debug("Module '%s' does not specify any build rules.\n", moduleName)
		return packages
	}

	err = filepath.Walk(rulesDirPath, func(filePath string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if file.IsDir() || path.Ext(file.Name()) != ".go" {
			return nil
		}

		relativeFilePath := strings.TrimPrefix(filePath, modulePath+"/")
		copyFilePath := path.Join(buildFilesDir, relativeFilePath)
		util.CopyFile(filePath, copyFilePath)
		return nil
	})

	if err != nil {
		log.Fatal("Failed to copy rule files for module '%s': %s.\n", moduleName, err)
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

func createModFileContent(moduleName string, modules map[string]string, pathPrefix string) []byte {
	mod := strings.Builder{}

	fmt.Fprintf(&mod, "module %s\n\n", moduleName)
	fmt.Fprintf(&mod, "go %s\n\n", goVersion)

	for modName := range modules {
		fmt.Fprintf(&mod, "require %s v0.0.0\n", modName)
		fmt.Fprintf(&mod, "replace %s => %s/%s\n\n", modName, pathPrefix, modName)
	}

	return []byte(mod.String())
}

func createGeneratorMainFile(generatorDir string, packages []string, modules map[string]string) {
	importLines := []string{}
	dbtMainLines := []string{}
	for idx, pkg := range packages {
		importLines = append(importLines, fmt.Sprintf("import p%d \"%s\"", idx, pkg))
		dbtMainLines = append(dbtMainLines, fmt.Sprintf("    merge(vars, p%d.DbtGetVariables())", idx))
	}

	mainFilePath := path.Join(generatorDir, mainFileName)
	mainFileContent := fmt.Sprintf(mainFileTemplate, strings.Join(importLines, "\n"), strings.Join(dbtMainLines, "\n"))
	util.WriteFile(mainFilePath, []byte(mainFileContent))

	modFilePath := path.Join(generatorDir, modFileName)
	modFileContent := createModFileContent("root", modules, ".")
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
