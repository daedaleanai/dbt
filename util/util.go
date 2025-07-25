package util

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/daedaleanai/dbt/v3/assets"
	"github.com/daedaleanai/dbt/v3/log"
	"gopkg.in/yaml.v2"
)

// ModuleFileName is the name of the file describing each module.
const (
	ModuleFileName      = "MODULE"
	ModuleSyntaxVersion = 3

	BuildDirName = "BUILD"
	// DepsDirName is directory that dependencies are stored in.
	DepsDirName     = "DEPS"
	WarningFileName = "WARNING.readme.txt"
)

const (
	fileMode = 0664
	dirMode  = 0775
)

// NOTE: We limit the integers to at most 6 digits.
var semVerRe = regexp.MustCompile(`v(\d{1,6})\.(\d{1,6})\.(\d{1,6})(-([\d\w.]+))?(\+[\d\w.]+)?`)

// Reimplementation of CutPrefix for backwards compatibility with versions < 1.20
func CutPrefix(str string, prefix string) (string, bool) {
	if strings.HasPrefix(str, prefix) {
		return str[len(prefix):], true
	}
	return str, false
}

// If -tags=semver-override=xxxxxx is specified among build info settings, then that one is used;
// otherwise Main.Version is used.
// If the version deduced by the algorithm above does not match semantic version format,
// the binary panics.
func obtainVersion() (string, uint, uint, uint) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		panic("The binary has been built without build information")
	}

	ver := bi.Main.Version

	for _, m := range bi.Settings {
		if m.Key != "-tags" {
			continue
		}
		for _, kv := range strings.Split(m.Value, ",") {
			kv := strings.TrimSpace(kv)
			if sfx, ok := CutPrefix(kv, "semver-override="); ok {
				ver = sfx
				break
			}

		}
	}

	const (
		base = 10
		bits = 20
	)

	m := semVerRe.FindStringSubmatch(ver)
	if len(m) > 0 && m[0] == ver {
		major, _ := strconv.ParseUint(m[1], base, bits)
		minor, _ := strconv.ParseUint(m[2], base, bits)
		patch, _ := strconv.ParseUint(m[3], base, bits)
		return m[0], uint(major), uint(minor), uint(patch)
	}

	panic("Could not determine DBT semantic version. " +
		"If compiling locally, compile with `go build -tags=semver-override=vX.Y.Z` for some X, Y and Z.")
}

func VersionTriplet() [3]uint {
	_, major, minor, patch := obtainVersion()
	return [3]uint{major, minor, patch}
}

func Version() string {
	s, _, _, _ := obtainVersion()
	return s
}

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

// MkdirAll creates directory `p` and all parent directories.
func MkdirAll(p string) {
	err := os.MkdirAll(p, dirMode)
	if err != nil {
		log.Fatal("Failed to create directory '%s': %s.\n", p, err)
	}
}

func ReadFile(filePath string) []byte {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal("Failed to read file '%s': %s.\n", filePath, err)
	}
	return data
}

func ReadJson(filePath string, v interface{}) {
	err := json.Unmarshal(ReadFile(filePath), v)
	if err != nil {
		log.Fatal("Failed to unmarshal JSON file '%s': %s.\n", filePath, err)
	}
}

func ReadYaml(filePath string, v interface{}) {
	err := yaml.Unmarshal(ReadFile(filePath), v)
	if err != nil {
		log.Fatal("Failed to unmarshal YAML file '%s': %s.\n", filePath, err)
	}
}

func WriteFile(filePath string, data []byte) {
	dir := path.Dir(filePath)
	err := os.MkdirAll(dir, dirMode)
	if err != nil {
		log.Fatal("Failed to create directory '%s': %s.\n", dir, err)
	}
	err = os.WriteFile(filePath, data, fileMode)
	if err != nil {
		log.Fatal("Failed to write file '%s': %s.\n", filePath, err)
	}
}

func GenerateFile(filePath string, tmpl template.Template, args any) {
	payload := bytes.Buffer{}
	if err := tmpl.Execute(&payload, args); err != nil {
		log.Fatal("Failed to generate file: %s: %w", filePath, err)
	}
	WriteFile(filePath, payload.Bytes())
}

func WriteJson(filePath string, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatal("Failed to marshal JSON for file '%s': %s.\n", filePath, err)
	}
	WriteFile(filePath, data)
}

func WriteYaml(filePath string, v interface{}) {
	data, err := yaml.Marshal(v)
	if err != nil {
		log.Fatal("Failed to marshal YAML for file '%s': %s.\n", filePath, err)
	}
	WriteFile(filePath, data)
}

func CopyFile(sourceFile, destFile string) {
	WriteFile(destFile, ReadFile(sourceFile))
}

// Copies a directory recursing into its inner directories
func CopyDirRecursively(sourceDir, destDir string) error {
	var wg sync.WaitGroup
	err := copyDirRecursivelyInner(sourceDir, destDir, &wg)
	wg.Wait()

	return err
}

func copyDirRecursivelyInner(sourceDir, destDir string, wg *sync.WaitGroup) error {
	stat, err := os.Stat(sourceDir)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("'%s' is not a directory", sourceDir)
	}

	MkdirAll(destDir)

	dirEntries, err := os.ReadDir(sourceDir)
	for _, dirEntry := range dirEntries {
		fileInfo, err := dirEntry.Info()
		if err != nil {
			return err
		}

		if dirEntry.IsDir() {
			err = copyDirRecursivelyInner(path.Join(sourceDir, dirEntry.Name()), path.Join(destDir, dirEntry.Name()), wg)
			if err != nil {
				return err
			}

			if err := os.Chmod(path.Join(destDir, dirEntry.Name()), fileInfo.Mode()); err != nil {
				return err
			}
		} else if (dirEntry.Type() & fs.ModeSymlink) == fs.ModeSymlink {
			linkTarget, err := os.Readlink(path.Join(sourceDir, dirEntry.Name()))

			if err != nil {
				return err
			}
			linkPath := path.Join(destDir, dirEntry.Name())

			if err = os.Symlink(linkTarget, linkPath); err != nil {
				return err
			}
		} else {
			wg.Add(1)

			go func(source, dest string, sourceFileInfo os.FileInfo) {
				defer wg.Done()
				CopyFile(source, dest)
				if err := os.Chmod(dest, sourceFileInfo.Mode()); err != nil {
					log.Fatal("failed to change filemode: ", err)
				}
			}(path.Join(sourceDir, dirEntry.Name()), path.Join(destDir, dirEntry.Name()), fileInfo)
		}
	}

	return nil
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
	root, err := getWorkspaceRoot()
	if err != nil {
		log.Fatal("Could not identify workspace root directory. Make sure you run this command inside a workspace: %s.\n", err)
	}

	return root
}

// internal version of getWorkspaceRoot that returns an error instead of aborting if we run dbt outside of a workspace.
func getWorkspaceRoot() (string, error) {
	var err error
	p := GetWorkingDir()
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

var FlagNoWorkspaceChecks = false

func CheckWorkspace() {
	if FlagNoWorkspaceChecks {
		return
	}

	workspaceRoot, err := getWorkspaceRoot()
	if err != nil {
		// No workspace to check
		return
	}

	checkManagedDir(workspaceRoot, BuildDirName)
	checkManagedDir(workspaceRoot, DepsDirName)
}

func checkManagedDir(root, child string) {
	if err := diagnoseExistingManagedDir(root, child); err != nil {
		log.Fatal("%vUse --no-workspace-checks to ignore this diagnostics.\n", err)
	}
}

func diagnoseExistingManagedDir(root, child string) error {
	dir := filepath.Join(root, child)
	stat, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Failed to stat directory %s: %w\n", dir, err)
	}
	if (stat.Mode() & os.ModeSymlink) != 0 {
		return fmt.Errorf("%s special directory must not be a symlink\n", child)
	}
	if !stat.IsDir() {
		return fmt.Errorf("Workspace contains file %s, which overlaps with a special purpose directory used by dbt\n", child)
	}
	return nil
}

func EnsureManagedDir(dir string) {
	workspaceRoot := GetWorkspaceRoot()
	if err := diagnoseExistingManagedDir(workspaceRoot, dir); err != nil {
		log.Warning("File or directory %s exists but it was modified outside of dbt."+
			" This is error-prone and shall be avoided.\n", dir)
		log.Warning("%v", err)
		return
	}

	if err := os.MkdirAll(filepath.Join(workspaceRoot, dir), dirMode); err != nil {
		log.Fatal("Failed to create special directory %s: %v", dir, err)
	}

	warningFilepath := filepath.Join(workspaceRoot, dir, WarningFileName)
	if _, err := os.Stat(warningFilepath); errors.Is(err, os.ErrNotExist) {
		// best effort, ignore errors
		if payload, err := assets.Statics.ReadFile("statics/WARNING.readme.txt"); err == nil {
			_ = os.WriteFile(warningFilepath, payload, fileMode)
		}
	}
}
