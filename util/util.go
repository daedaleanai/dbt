package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

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

// RemoveDir removes a directory and all of its content.
func RemoveDir(p string) {
	err := os.RemoveAll(p)
	if err != nil {
		log.Fatal("Failed to remove directory '%s': %s.\n", p, err)
	}
}

func ReadFile(filePath string) []byte {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file '%s': %s.\n", filePath, err)
	}
	return data
}

func WriteFile(filePath string, data []byte) {
	dir := path.Dir(filePath)
	err := os.MkdirAll(dir, DirMode)
	if err != nil {
		log.Fatal("Failed to create directory '%s': %s.\n", dir, err)
	}
	err = ioutil.WriteFile(filePath, data, FileMode)
	if err != nil {
		log.Fatal("Failed to write file '%s': %s.\n", filePath, err)
	}
}

func CopyFile(sourceFile, destFile string) {
	WriteFile(destFile, ReadFile(sourceFile))
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

func GetModuleRootForPath(p string) string {
	moduleRoot, err := getModuleRoot(p)
	if err != nil {
		log.Fatal("Could not identify module root directory. Make sure you run this command inside a module: %s.\n", err)
	}
	return moduleRoot
}

// GetModuleRoot returns the root directory of the current module.
func GetModuleRoot() string {
	workingDir := GetWorkingDir()
	moduleRoot, err := getModuleRoot(workingDir)
	if err != nil {
		log.Fatal("Could not identify module root directory. Make sure you run this command inside a module: %s.\n", err)
	}
	return moduleRoot
}

// GetWorkspaceRoot returns the root directory of the current workspace (i.e., top-level module).
func GetWorkspaceRoot() string {
	var err error
	p := GetWorkingDir()
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

// GetWorkingDir returns the current working directory.
func GetWorkingDir() string {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get working directory: %s.\n", err)
	}
	return workingDir
}

// WalkSymlink works like filepath.Walk but also accepts symbolic links as `root`.
func WalkSymlink(root string, walkFn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}

	if (info.Mode() & os.ModeSymlink) != os.ModeSymlink {
		return filepath.Walk(root, walkFn)
	}

	link, err := os.Readlink(root)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(link) {
		link = path.Join(path.Dir(root), link)
	}

	return filepath.Walk(link, func(file string, info os.FileInfo, err error) error {
		file = path.Join(root, strings.TrimPrefix(file, link))
		return walkFn(file, info, err)
	})
}
