package cmd

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/daedaleanai/dbt/v3/log"
	"github.com/daedaleanai/dbt/v3/lsp"
	"github.com/daedaleanai/dbt/v3/module"
	"github.com/daedaleanai/dbt/v3/util"

	"github.com/daedaleanai/cobra"

	"golang.org/x/tools/go/packages"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Args:  cobra.ArbitraryArgs,
	Short: "Executes a gopackagedriver to support LSP completion for dbt sources",
	Long:  `Executes a gopackagedriver to support LSP completion for dbt sources.`,
	Run: func(cmd *cobra.Command, args []string) {
		runLsp(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
}

type lspGoFile struct {
	// Absolute path to the source file
	SourcePath string

	// Relatve path from go root
	CopyPath string

	// Imports by package path found within the file
	Imports []string

	// Name of the package parsed from the source code
	PackageName string
}

type lspModuleData struct {
	moduleName string
	buildFiles []lspGoFile
	ruleFiles  []lspGoFile
	initFiles  []lspGoFile
}

func lspProcessGoFile(overlays map[string][]byte, file *lspGoFile) error {
	var src any
	if innerData, ok := overlays[file.SourcePath]; ok {
		// If there is an overlay we should take it
		src = innerData
	}

	parsedFile, err := parser.ParseFile(token.NewFileSet(), file.SourcePath, src, parser.ImportsOnly)
	if err != nil {
		return err
	}

	// Package name parsed from the source file. Unfortunately, in go this is allowed to differ from
	// the last component of the import path
	file.PackageName = parsedFile.Name.String()

	for _, imp := range parsedFile.Imports {
		file.Imports = append(file.Imports, strings.Trim(imp.Path.Value, "\""))
	}
	return nil
}

func lspProcessModules(request *packages.DriverRequest, generatorDir string, mods []*lspModuleData) error {
	for _, mod := range mods {
		for i := range mod.ruleFiles {
			if err := lspProcessGoFile(request.Overlay, &mod.ruleFiles[i]); err != nil {
				return err
			}
		}

		for i := range mod.buildFiles {
			if err := lspProcessGoFile(request.Overlay, &mod.buildFiles[i]); err != nil {
				return err
			}

			packagePath := filepath.Dir(mod.buildFiles[i].CopyPath)
			initFile := lspGoFile{
				SourcePath: filepath.Join(generatorDir, packagePath, "init.go"),
				CopyPath:   filepath.Join(packagePath, "init.go"),
			}
			if err := lspProcessGoFile(request.Overlay, &initFile); err != nil {
				return err
			}
			mod.initFiles = append(mod.initFiles, initFile)
		}

	}
	return nil
}

func lspPackagesFromModules(mods []*lspModuleData) map[string]*lsp.Package {
	packagesByImportPath := make(map[string]*lsp.Package)
	for _, mod := range mods {
		allGoFilesInMod := slices.Clone(mod.buildFiles)
		allGoFilesInMod = append(allGoFilesInMod, mod.ruleFiles...)
		allGoFilesInMod = append(allGoFilesInMod, mod.initFiles...)

		for _, goFile := range allGoFilesInMod {
			importPath := filepath.Dir(goFile.CopyPath)

			if _, ok := packagesByImportPath[importPath]; !ok {
				packagesByImportPath[importPath] = &lsp.Package{
					// Taken from the first file in this import path
					Name:       goFile.PackageName,
					ImportPath: importPath,
					GoFiles:    []string{},
				}
			}

			packagesByImportPath[importPath].GoFiles = append(packagesByImportPath[importPath].GoFiles, goFile.SourcePath)
			for _, imp := range goFile.Imports {
				if !slices.Contains(packagesByImportPath[importPath].Imports, imp) {
					packagesByImportPath[importPath].Imports = append(packagesByImportPath[importPath].Imports, imp)
				}
			}
		}
	}

	return packagesByImportPath

}

func runLsp(_ *cobra.Command, args []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	dbtRulesDir := filepath.Join(workspaceRoot, util.DepsDirName, dbtRulesDirName)
	if !util.DirExists(dbtRulesDir) {
		log.Fatal("You are running 'dbt lsp' without '%s' being available. Add that dependency, run 'dbt sync' and try again.\n", dbtRulesDirName)
		return
	}

	util.EnsureManagedDir(util.BuildDirName)

	// Read request from stdin
	reqData, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal("Error reading request from stdin: %v", err)
	}
	log.Debug("Request: %s\n", string(reqData))

	var request packages.DriverRequest
	if err := json.Unmarshal(reqData, &request); err != nil {
		log.Fatal("Error decoding driver request: %v", err)
	}

	// We need the generator to be avalable so that generated files can also participate in package
	// resolution completions
	generatorDir := populateGenerator()

	var modData []*lspModuleData

	modules := module.GetAllModules(workspaceRoot)
	for _, modEntry := range modules.Entries() {
		mod := modEntry.Value

		var rules []lspGoFile
		for _, rule := range module.ListRules(mod) {
			rules = append(rules, lspGoFile{
				CopyPath:   rule.CopyPath,
				SourcePath: rule.SourcePath,
				Imports:    nil,
			})
		}
		var buildFiles []lspGoFile
		for _, buildFile := range module.ListBuildFiles(mod) {
			buildFiles = append(buildFiles, lspGoFile{
				CopyPath:   buildFile.CopyPath,
				SourcePath: buildFile.SourcePath,
				Imports:    nil,
			})
		}

		modData = append(modData, &lspModuleData{
			moduleName: modEntry.Key,
			ruleFiles:  rules,
			buildFiles: buildFiles,
		})
	}

	lspProcessModules(&request, generatorDir, modData)
	pkgs := lspPackagesFromModules(modData)

	lspDriver := lsp.NewDriver(pkgs)

	response, err := lspDriver.HandleRequest(&request, args)
	if err != nil {
		log.Fatal("Error handling request: %v", err)
	}

	rsp, err := json.Marshal(response)
	if err != nil {
		log.Fatal("Error encoding response: %v", err)
	}

	log.Debug("Response: %s\n", string(rsp))

	if _, err := os.Stdout.Write(rsp); err != nil {
		log.Fatal("Error writing response: %v", err)
	}
}
