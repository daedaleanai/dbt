package cmd

import (
	"bytes"
	"dwm/log"
	"dwm/module"
	"dwm/util"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Recursively clones and updates modules to satisfy all dependencies.",
	Long: `Recursively clones and updates modules to satisfy the dependencies
declared in the MODULE files of each module, starting from the top-level MODULE file.
`,
	Run: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func createModule(modulePath string, url string) (module.Module, error) {
	log.Spinner.Start()
	defer log.Spinner.Stop()

	if strings.HasSuffix(url, ".git") {
		log.Log(2, "Cloning '%s'.\n", url)
		return module.CreateGitModule(modulePath, url)
	}

	if strings.HasSuffix(url, ".tar.gz") {
		log.Log(2, "Downloading '%s'.\n", url)
		return module.CreateTarModule(modulePath, url)
	}

	return nil, fmt.Errorf("could not determine module type from dependency url '%s'", url)
}

func setupModule(mod module.Module) {
	setupFilePath := path.Join(mod.Path(), util.SetupFileName)
	if !util.FileExists(setupFilePath) {
		log.Log(2, "Module has no SETUP.go file.\n")
		return
	}

	log.Log(2, "Running SETUP.go.\n")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", setupFilePath)
	cmd.Dir = mod.Path()
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		log.Error(2, "Running SETUP.go failed:\nSTDOUT:\n%s\n\nSTDERR:\n%s\n", string(stdout.Bytes()), string(stderr.Bytes()))
	}
}

func createOrOpenModule(modulePath string, url string) (module.Module, error) {
	if util.DirExists(modulePath) {
		return module.OpenModule(modulePath)
	}

	mod, err := createModule(modulePath, url)
	if err != nil {
		return nil, err
	}

	setupModule(mod)
	return mod, nil
}

func getModuleDependencies(mod module.Module) []module.Dependency {
	moduleFilePath := path.Join(mod.Path(), util.ModuleFileName)
	if !util.FileExists(moduleFilePath) {
		log.Warn(1, "Module has no MODULE file.\n")
		return []module.Dependency{}
	}
	deps, err := module.ReadModuleFile(moduleFilePath)
	if err != nil {
		log.Error(1, "Failed to read MODULE file: %s.\n", err)
	}
	return deps
}

func runSync(cmd *cobra.Command, args []string) {
	workspaceRoot, err := util.GetWorkspaceRoot()
	if err != nil {
		log.Error(0, "Could not identify workspace root directory: %s.\n", err)
	}
	log.Log(0, "Current workspace is '%s'.\n", workspaceRoot)

	// The top-level module must already exist and will never be cloned or downloaded by the tool.
	rootModule, err := module.OpenModule(workspaceRoot)
	if err != nil {
		log.Error(0, "Failed open top-level module: %s.\n", err)
	}

	// Create the DEPS subdirectory and create a symlink to the top-level module.
	rootModuleSymlink := path.Join(workspaceRoot, util.DepsDirName, rootModule.Name())
	if !util.DirExists(rootModuleSymlink) {
		setupModule(rootModule)

		log.Log(0, "Creating symlink for the top-level module: DEPS/%s -> %s.\n", rootModule.Name(), workspaceRoot)

		err = os.MkdirAll(path.Dir(rootModuleSymlink), util.DirMode)
		if err != nil {
			log.Error(0, "Failed to create DEPS directory: %s.\n", err)
		}
		err = os.Symlink("..", rootModuleSymlink)
		if err != nil {
			log.Error(0, "Failed to create symlink to top-level module: %s.\n", err)
		}

	}

	// Keeps track of the modules whose versions have already been fixed and the
	// dependent module that caused the version to be fixed.
	fixed := map[string]string{rootModule.Name(): rootModule.Name()}

	// Modules that still need to be processed.
	queue := []module.Module{rootModule}

	for len(queue) > 0 {
		mod := queue[0]
		queue = queue[1:]

		log.Log(0, "Processing module '%s'.\n", mod.Name())

		deps := getModuleDependencies(mod)
		log.Log(1, "Module has %d dependencies.\n", len(deps))

		for idx, dep := range deps {
			log.Log(1, "%d) Resolving dependency to '%s', version '%s'.\n", idx+1, dep.ModuleName(), dep.Version)

			depPath := path.Join(workspaceRoot, util.DepsDirName, dep.ModuleName())
			depMod, err := createOrOpenModule(depPath, dep.URL)
			if err != nil {
				log.Error(2, "Failed to open module: %s.\n", err)
			}

			// If the version of the dependency is not yet fixed, the current module will determine
			// the dependency version.
			dependentModule, versionIsFixed := fixed[depMod.Name()]
			if !versionIsFixed {
				fixed[depMod.Name()] = mod.Name()
				queue = append(queue, depMod)
			}

			// Check that the module fulfilling dependency actually comes from source required
			// by the dependency. I.e., that the dependency module is not a different module that
			// just happens to have the same name.
			hasRemote, err := depMod.HasRemote(dep.URL)
			if err != nil {
				log.Error(2, "Failed to check dependency URL: %s.\n", err)
			}
			if !hasRemote {
				log.Warn(2, "Module origin does not match dependency URL '%s'.\n", dep.URL)
			}

			// If the dependency module has uncommited changes, don't try to change its version.
			isDirty, err := depMod.IsDirty()
			if err != nil {
				log.Error(2, "Failed to read module status: %s.\n", err)
			}
			if isDirty {
				log.Warn(2, "Module is in a dirty state. Not changing version.\n")
				continue
			}

			// If the dependency module already has the required version checked out, there is nothing to do.
			hasCorrectVersion, err := depMod.HasVersionCheckedOut(dep.Version)
			if err != nil {
				log.Error(2, "Failed to check current module version: %s.\n", err)
			}
			if hasCorrectVersion {
				log.Success(2, "Module version matches required version.\n")
				continue
			}

			// If the dependency module's version is already fixed, but does not match
			// the version required by the dependency, we issue a warning but do not change the version.
			if versionIsFixed {
				log.Warn(2, "Module version is already fixed by dependent module '%s'. Not changing version.\n", dependentModule)
				continue
			}

			log.Log(2, "Changing version to '%s'.\n", dep.Version)
			err = depMod.CheckoutVersion(dep.Version)
			if err != nil {
				log.Error(2, "Failed to checkout version '%s': %s.\n", dep.Version, err)
			}

			// Verify that changing the version has worked.
			hasCorrectVersion, err = depMod.HasVersionCheckedOut(dep.Version)
			if err != nil {
				log.Error(2, "Failed to check current module version: %s.\n", err)
			}
			if !hasCorrectVersion {
				log.Error(2, "Failed to check out required module version '%s'.\n", dep.Version)
			}
		}

		log.Log(2, "\n")
	}

	log.Success(0, "Done.\n")
}
