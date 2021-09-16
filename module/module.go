package module

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

const setupFileName = "SETUP.go"
const setupSentinelFileName = ".setup"

// Module represents a checked-out module.
type Module interface {
	URL() string

	Head() string
	RevParse(rev string) string
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
		SetupModule(modulePath)
		return module
	}
	if strings.HasSuffix(url, ".tar.gz") {
		log.Debug("Module URL ends in '.tar.gz'. Trying to create a new TarModule.\n")
		module, err := createTarModule(modulePath, url)
		if err != nil {
			os.RemoveAll(modulePath)
			log.Fatal("Failed to create tar module: %s.\n", err)
		}
		SetupModule(modulePath)
		return module
	}

	log.Fatal("Failed to determine module type from dependency url '%s'.\n", url)
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
