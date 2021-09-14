package module

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

const setupFileName = "SETUP.go"
const setupSentinelFileName = ".setup"

// Module represents a checked-out module.
type Module interface {
	Name() string
	Path() string
	URL() string

	Head() string
	RevParse(ref string) string
	IsDirty() bool

	Fetch() bool
	Checkout(hash string)
}

// OpenModule opens a module checked out on disk.
func OpenModule(modulePath string) Module {
	log.Debug("Opening module '%s'.\n", modulePath)

	if util.DirExists(path.Join(modulePath, ".git")) {
		log.Debug("Found '.git' directory. Expecting this to be a GitModule.\n")
		return GitModule{modulePath}
	}

	if util.FileExists(path.Join(modulePath, tarMetadataFileName)) {
		log.Debug("Found '%s' file. Expecting this to be a TarModule.\n", tarMetadataFileName)
		return TarModule{modulePath}
	}

	log.Fatal("Module appears to be broken. Remove the module directory and rerun 'dbt sync'.\n")
	return nil
}

// OpenOrCreateModule tries to open the module in `modulePath`. If the `modulePath` directory does
// not yet exists, it tries to create a new module by cloning / downloading the module from `url`.
func OpenOrCreateModule(modulePath string, url string) Module {
	log.Debug("Opening or creating module '%s' from url '%s'.\n", modulePath, url)
	if util.DirExists(modulePath) {
		log.Debug("Module directory exists.\n")
		return OpenModule(modulePath)
	}

	log.Debug("Module directory does not exists.\n")

	if strings.HasSuffix(url, ".git") {
		log.Debug("Module URL ends in '.git'. Trying to create a new git module.\n")
		module, err := createGitModule(modulePath, url)
		if err != nil {
			os.RemoveAll(modulePath)
			log.Fatal("Failed to create git module: %s.\n", err)
		}
		return module
	}
	if strings.HasSuffix(url, ".tar.gz") {
		log.Debug("Module URL ends in '.tar.gz'. Trying to create a new TarModule.\n")
		module, err := createTarModule(modulePath, url)
		if err != nil {
			os.RemoveAll(modulePath)
			log.Fatal("Failed to create tar module: %s.\n", err)
		}
		return module
	}

	log.Fatal("Failed to determine module type from dependency url '%s'.\n", url)
	return nil
}

// SetupModule runs the SETUP.go file in the root directory of `mod` (it if exists).
func SetupModule(mod Module) {
	log.Debug("Trying to set up module '%s'.\n", mod.Name())

	setupSentinelFilePath := path.Join(mod.Path(), setupSentinelFileName)
	if util.FileExists(setupSentinelFilePath) {
		log.Debug("%s file already exists in module directory. Nothing to do.\n", setupSentinelFileName)
		return
	}

	setupFilePath := path.Join(mod.Path(), setupFileName)
	if !util.FileExists(setupFilePath) {
		log.Debug("Module has no SETUP.go file. Nothing to do.\n")
		return
	}

	log.Log("Running 'go run %s'.\n", setupFilePath)
	cmd := exec.Command("go", "run", setupFilePath)
	cmd.Dir = mod.Path()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Fatal("Running SETUP.go failed: %s.\n")
	}
	log.Success("Module is set up.\n")

	log.Debug("Creating sentinel file.\n")
	util.WriteFile(setupSentinelFilePath, []byte{})
}

// GetAllModulePaths returns all the names and paths of all modules in the workspace.
func GetAllModulePaths(workspaceRoot string) map[string]string {
	depsDir := path.Join(workspaceRoot, util.DepsDirName)
	if !util.DirExists(depsDir) {
		log.Warning("There is no %s/ directory in the workspace. Try running 'dbt sync' first.\n", util.DepsDirName)
		return nil
	}
	files, err := ioutil.ReadDir(depsDir)
	if err != nil {
		log.Fatal("Failed to read content of %s/ directory: %s.\n", util.DepsDirName, err)
	}
	modules := map[string]string{}
	for _, file := range files {
		if file.IsDir() || (file.Mode()&os.ModeSymlink) == os.ModeSymlink {
			modules[file.Name()] = path.Join(depsDir, file.Name())
		}
	}
	return modules
}
