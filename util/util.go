package util

import (
	"fmt"
	"os"
	"path"
)

// FileMode is the default FileMode used when creating files.
const FileMode = 0664

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
		parentDirName := path.Base(path.Dir(p))
		if FileExists(moduleFilePath) || parentDirName == DepsDirName {
			return p, nil
		}
		if p == "/" {
			return "", fmt.Errorf("not inside a module")
		}
		p = path.Dir(p)
	}
}

// GetModuleRoot returns the root directory of the current module.
func GetModuleRoot() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return getModuleRoot(workingDir)
}

// GetWorkspaceRoot returns the root directory of the current workspace (i.e., top-level module).
func GetWorkspaceRoot() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	p := workingDir
	for {
		p, err = getModuleRoot(p)
		if err != nil {
			return "", err
		}

		parentDirName := path.Base(path.Dir(p))
		if parentDirName != DepsDirName {
			return p, nil
		}
		p = path.Dir(p)
	}
}
