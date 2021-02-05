package util

import (
	"fmt"
	"os"
	"path"

	"github.com/daedaleanai/dbt/log"
)

// FileMode is the default FileMode used when creating files.
const FileMode = 0664

// DirMode is the default FileMode used when creating directories.
const DirMode = 0775

// ModuleFileName is the name of the file describing each module.
const ModuleFileName = "MODULE"

// DepsDirName is directory that dependencies are stored in.
const DepsDirName = "DEPS"

// FileExists checks whether some file exists.
func FileExists(file string) bool {
	stat, err := os.Stat(file)
	return err == nil && !stat.IsDir()
}

// DirExists checks whether some directory exists.
func DirExists(dir string) bool {
	stat, err := os.Stat(dir)
	return err == nil && stat.IsDir()
}

func getModuleRoot(p string) (string, error) {
	for {
		moduleFilePath := path.Join(p, ModuleFileName)
		gitDirPath := path.Join(p, ".git")
		parentDirName := path.Base(path.Dir(p))
		if FileExists(moduleFilePath) || parentDirName == DepsDirName || DirExists(gitDirPath) {
			return p, nil
		}
		if p == "/" {
			return "", fmt.Errorf("not inside a module")
		}
		p = path.Dir(p)
	}
}

// GetModuleRoot returns the root directory of the current module.
func GetModuleRoot() string {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get working directory: %s.\n", err)
	}
	moduleRoot, err := getModuleRoot(workingDir)
	if err != nil {
		log.Fatal("Could not identify module root directory. Make sure you run this command inside a module: %s.\n", err)
	}
	return moduleRoot
}

// GetWorkspaceRoot returns the root directory of the current workspace (i.e., top-level module).
func GetWorkspaceRoot() string {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get working directory: %s.\n", err)
	}

	p := workingDir
	for {
		p, err = getModuleRoot(p)
		if err != nil {
			log.Fatal("Could not identify workspace root directory. Make sure you run this command inside a workspace: %s.\n", err)
		}

		parentDirName := path.Base(path.Dir(p))
		if parentDirName != DepsDirName {
			return p
		}
		p = path.Dir(p)
	}
}
