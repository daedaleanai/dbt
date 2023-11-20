package module

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/daedaleanai/dbt/v3/log"
	"github.com/daedaleanai/dbt/v3/util"
)

const setupFileName = "SETUP.go"
const setupSentinelFileName = ".setup"
const rulesDirName = "RULES"
const buildFileName = "BUILD.go"

type GoFile struct {
	// Absolute path to file
	SourcePath string
	// Relatve path from go root
	CopyPath string
}

type GoModule struct {
	// Module name
	Name string
	// Used modules
	Deps []string
}

// Module represents a checked-out module.
type Module interface {
	Name() string
	URL() string

	Head() string
	RevParse(rev string) string
	IsDirty() bool
	IsAncestor(ancestor, rev string) bool

	Fetch() bool
	Checkout(hash string)

	RootPath() string
	Type() ModuleType
}

func orderedGoFiles(files []GoFile) []GoFile {
	return util.SliceOrderedBy(files, func(f *GoFile) string { return f.SourcePath })
}

func listGoModules(module Module, moduleFile ModuleFile) []GoModule {
	modulePath := module.RootPath()
	moduleName := path.Base(modulePath) // FIXME: interface method

	return []GoModule{
		{
			Name: moduleName,
			Deps: util.OrderedKeys(moduleFile.Dependencies),
		},
	}
}

func listGoModulesCpp(module Module, moduleFile ModuleFile) []GoModule {
	modulePath := module.RootPath()
	moduleName := path.Base(modulePath) // FIXME: interface method

	deps := util.OrderedKeys(moduleFile.Dependencies)

	result := []GoModule{}
	result = append(result, GoModule{
		Name: moduleName,
		Deps: deps,
	})

	rulesDirPath := path.Join(modulePath, rulesDirName)

	if !util.DirExists(rulesDirPath) {
		return result
	}

	files, err := ioutil.ReadDir(rulesDirPath)
	if err != nil {
		log.Fatal("Failed to read content of %s/ directory: %s.\n", rulesDirPath, err)
	}

	for _, subdirName := range util.OrderedSlice(util.MappedSlice(files, func(fi os.FileInfo) string { return fi.Name() })) {
		result = append(result, GoModule{
			Name: subdirName,
			Deps: deps,
		})
	}
	return result
}

func ListGoModules(module Module) []GoModule {
	moduleFile := ReadModuleFile(module.RootPath())
	if moduleFile.Layout == "cpp" {
		return listGoModulesCpp(module, moduleFile)
	} else {
		return listGoModules(module, moduleFile)
	}
}

func listBuildFiles(module Module) []GoFile {
	modulePath := module.RootPath()
	result := []GoFile{}
	moduleName := path.Base(modulePath) // FIXME: interface method
	err := util.WalkSymlink(modulePath, func(filePath string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativeFilePath := strings.TrimPrefix(filePath, modulePath+"/")

		// Ignore the BUILD/, DEPS/ and RULES/ directories.
		if file.IsDir() && (relativeFilePath == util.BuildDirName || relativeFilePath == util.DepsDirName || relativeFilePath == rulesDirName) {
			return filepath.SkipDir
		}

		// Skip everything that is not a BUILD.go file.
		if file.IsDir() || file.Name() != buildFileName {
			return nil
		}

		log.Debug("Found %s file '%s'.\n", buildFileName, path.Join(modulePath, relativeFilePath))

		relativeFilePath = path.Join(moduleName, relativeFilePath)

		result = append(result, GoFile{
			SourcePath: filePath,
			CopyPath:   relativeFilePath,
		})
		return nil
	})
	if err != nil {
		log.Fatal("Failed to process %s files for module %s: %s.\n", buildFileName, moduleName, err)
	}

	return orderedGoFiles(result)
}

func listBuildFilesCpp(module Module) []GoFile {
	modulePath := module.RootPath()
	result := []GoFile{}
	moduleName := path.Base(modulePath) // FIXME: interface method
	err := util.WalkSymlink(modulePath, func(filePath string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativeFilePath := strings.TrimPrefix(filePath, modulePath+"/")

		// Ignore the BUILD/, DEPS/ and RULES/ directories.
		if file.IsDir() && (relativeFilePath == util.BuildDirName || relativeFilePath == util.DepsDirName || relativeFilePath == rulesDirName) {
			return filepath.SkipDir
		}

		// Skip everything that is not a BUILD.go file.
		if file.IsDir() || file.Name() != buildFileName {
			return nil
		}

		parts := strings.Split(relativeFilePath, "/")
		if len(parts) <= 2 {
			return nil
		}

		relativeFilePath = strings.Join(parts[1:], "/")

		log.Debug("Found %s file '%s'.\n", buildFileName, path.Join(modulePath, relativeFilePath))

		result = append(result, GoFile{
			SourcePath: filePath,
			CopyPath:   relativeFilePath,
		})
		return nil
	})
	if err != nil {
		log.Fatal("Failed to process %s files for module %s: %s.\n", buildFileName, moduleName, err)
	}

	return orderedGoFiles(result)
}

func listRules(module Module) []GoFile {
	modulePath := module.RootPath()
	rulesDirPath := path.Join(modulePath, rulesDirName)
	result := []GoFile{}
	moduleName := path.Base(modulePath) // FIXME: interface method

	if !util.DirExists(rulesDirPath) {
		return result
	}

	err := filepath.Walk(rulesDirPath, func(filePath string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if file.IsDir() || path.Ext(file.Name()) != ".go" {
			return nil
		}

		relativeFilePath := strings.TrimPrefix(filePath, path.Dir(modulePath)+"/")

		result = append(result, GoFile{
			SourcePath: filePath,
			CopyPath:   relativeFilePath,
		})
		return nil
	})
	if err != nil {
		log.Fatal("Failed to process %s/ files for module '%s': %s.\n", rulesDirName, moduleName, err)
	}

	return orderedGoFiles(result)
}

func listRulesCpp(module Module) []GoFile {
	modulePath := module.RootPath()
	rulesDirPath := path.Join(modulePath, rulesDirName)
	result := []GoFile{}
	moduleName := path.Base(modulePath) // FIXME: interface method

	if !util.DirExists(rulesDirPath) {
		return result
	}

	err := filepath.Walk(rulesDirPath, func(filePath string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if file.IsDir() || path.Ext(file.Name()) != ".go" {
			return nil
		}

		relativeFilePath := strings.TrimPrefix(filePath, modulePath+"/")
		parts := strings.Split(relativeFilePath, "/")
		if len(parts) <= 2 {
			return nil
		}

		name := parts[1]
		parts[1] = rulesDirName
		parts[0] = name

		relativeFilePath = strings.Join(parts, "/")

		result = append(result, GoFile{
			SourcePath: filePath,
			CopyPath:   relativeFilePath,
		})
		return nil
	})
	if err != nil {
		log.Fatal("Failed to process %s/ files for module '%s': %s.\n", rulesDirName, moduleName, err)
	}

	return orderedGoFiles(result)
}

func ListRules(module Module) []GoFile {
	moduleFile := ReadModuleFile(module.RootPath())
	if moduleFile.Layout == "cpp" {
		return listRulesCpp(module)
	} else {
		return listRules(module)
	}
}

func ListBuildFiles(module Module) []GoFile {
	moduleFile := ReadModuleFile(module.RootPath())
	if moduleFile.Layout == "cpp" {
		return listBuildFilesCpp(module)
	} else {
		return listBuildFiles(module)
	}
}

// OpenModuleByName opens a module checked out on disk.
func OpenModuleByName(moduleName string) Module {
	wsRoot := util.GetWorkspaceRoot()
	workspaceModuleFile := ReadModuleFile(wsRoot)
	if workspaceModuleFile.Layout == "cpp" && moduleName == path.Base(wsRoot) {
		// The root module in cpp layout is not symlinked
		return OpenModule(wsRoot)
	}

	// The module is in the deps folder
	depsDir := path.Join(wsRoot, util.DepsDirName)
	modPath := path.Join(depsDir, moduleName)
	return OpenModule(modPath)
}

// OpenModule opens a module checked out on disk.
func OpenModule(modulePath string) Module {
	log.Debug("Opening module '%s'.\n", modulePath)

	if util.DirExists(path.Join(modulePath, ".git")) {
		log.Debug("Found '.git' directory. Expecting this to be a GitModule.\n")
		module := GitModule{path: modulePath}
		mirror, _ := getOrCreateGitMirror(module.URL())
		return GitModule{path: modulePath, mirror: mirror}
	}

	if util.FileExists(path.Join(modulePath, ".git")) {
		log.Debug("Found '.git' worktree file. Expecting this to be a GitModule.\n")
		module := GitModule{path: modulePath}
		mirror, _ := getOrCreateGitMirror(module.URL())
		return GitModule{path: modulePath, mirror: mirror}
	}

	if util.FileExists(path.Join(modulePath, tarMetadataFileName)) {
		log.Debug("Found '%s' file. Expecting this to be a TarModule.\n", tarMetadataFileName)
		module := TarModule{path: modulePath}
		mirror, _ := getOrCreateTarMirror(module.URL())
		return TarModule{path: modulePath, mirror: mirror}
	}

	log.Fatal("Module appears to be broken. Remove the module directory and rerun 'dbt sync'.\n")
	return nil
}

type ModuleType uint

const (
	GitModuleType ModuleType = iota
	TarGzModuleType
)

func (t ModuleType) String() string {
	switch t {
	case GitModuleType:
		return "git"
	case TarGzModuleType:
		return "tar.gz"
	}

	log.Fatal("Invalid module type: %s\n", t)
	return ""
}

func ParseModuleTypeString(str string) (ModuleType, bool) {
	if str == "git" {
		return GitModuleType, true
	} else if str == "tar.gz" {
		return TarGzModuleType, true
	}

	return GitModuleType, false
}

func DetermineModuleType(url, moduleTypeString string) ModuleType {
	if moduleType, ok := ParseModuleTypeString(moduleTypeString); ok {
		return moduleType
	}

	if moduleTypeString != "" {
		log.Fatal("Invalid module type '%s'.\n", moduleTypeString)
	}

	if strings.HasSuffix(url, ".git") {
		log.Debug("Module URL ends in '.git'. Trying to create a new git module.\n")
		return GitModuleType
	}
	if strings.HasSuffix(url, ".tar.gz") {
		log.Debug("Module URL ends in '.tar.gz'. Trying to create a new TarModule.\n")
		return TarGzModuleType
	}

	log.Fatal("Failed to determine module type from dependency url '%s'.\n", url)
	return GitModuleType // Just because golang is not clever enough to notice that this is unreachable.
}

// OpenOrCreateModule tries to open the module in `modulePath`. If the `modulePath` directory does
// not yet exists, it tries to create a new module by cloning / downloading the module from `url`.
func OpenOrCreateModule(modulePath string, url string, moduleTypeString string, expectedHash string) Module {
	log.Debug("Opening or creating module '%s' from url '%s'.\n", modulePath, url)
	if util.DirExists(modulePath) {
		log.Debug("Module directory exists.\n")
		return OpenModule(modulePath)
	}

	log.Debug("Module directory does not exists.\n")

	moduleType := DetermineModuleType(url, moduleTypeString)

	if moduleType == GitModuleType {
		module, err := CreateGitModule(modulePath, url)
		if err != nil {
			os.RemoveAll(modulePath)
			log.Fatal("Failed to create git module: %s.\n", err)
		}
		if module.Head() == expectedHash {
			SetupModule(modulePath)
		}
		return module
	} else if moduleType == TarGzModuleType {
		module, err := createTarModule(modulePath, url)
		if err != nil {
			os.RemoveAll(modulePath)
			log.Fatal("Failed to create tar module: %s.\n", err)
		}
		SetupModule(modulePath)
		return module
	}

	log.Fatal("Unhandled module type %v\n", moduleType)
	return nil
}

// SetupModule runs the SETUP.go file in the root directory of `mod` (it if exists).
func SetupModule(modulePath string) {
	setupFilePath := path.Join(modulePath, setupFileName)
	if !util.FileExists(setupFilePath) {
		log.Debug("Module has no %s file. Nothing to do.\n", setupFileName)
		return
	}

	log.Debug("Running 'go run %s'.\n", setupFilePath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.Command("go", "run", setupFilePath)
	cmd.Dir = modulePath
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		log.Fatal("Running %s timed out: %s.\n: %s.\n", setupFileName, ctx.Err())
	}
	if err != nil {
		log.Fatal("Running %s failed: %s.\n", setupFileName, err)
	}
	log.Success("Module is set up.\n")
}

// GetAllModules return all the names and modules in the workspace
func GetAllModules(workspaceRoot string) util.OrderedMap[string, Module] {
	depsDir := path.Join(workspaceRoot, util.DepsDirName)
	if !util.DirExists(depsDir) {
		log.Warning("There is no %s/ directory in the workspace. Try running 'dbt sync' first.\n", util.DepsDirName)
		return util.NewOrderedMap[string, Module]()
	}
	files, err := ioutil.ReadDir(depsDir)
	if err != nil {
		log.Fatal("Failed to read content of %s/ directory: %s.\n", util.DepsDirName, err)
	}
	modules := map[string]Module{}

	for _, file := range files {
		if file.IsDir() || (file.Mode()&os.ModeSymlink) == os.ModeSymlink {
			modules[file.Name()] = OpenModule(path.Join(depsDir, file.Name()))
		}
	}

	moduleFile := ReadModuleFile(workspaceRoot)
	if moduleFile.Layout == "cpp" {
		modules[path.Base(workspaceRoot)] = OpenModule(workspaceRoot)
	}

	return util.NewOrderedMapFrom(modules)
}
