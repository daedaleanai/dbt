package rules

import (
	"embed"
	"os"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/v3/log"
)

//go:embed rule_fs
var rules embed.FS

const embeddedFsDir string = "rule_fs"
const rulesDir string = embeddedFsDir + string(os.PathSeparator) + "RULES"

func listFilesImpl(dir string, filter func(string) bool, files *[]string) {
	entries, err := rules.ReadDir(dir)
	if err != nil {
		log.Fatal("Unable to read embedded dbt-rules/RULES directory: ", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			listFilesImpl(path.Join(dir, entry.Name()), filter, files)
		} else {
			if filter(entry.Name()) {
				embeddedPath := path.Join(dir, entry.Name())
				*files = append(*files, strings.TrimPrefix(embeddedPath, embeddedFsDir+string(os.PathSeparator)))
			}
		}
	}
}

func ListRuleFiles() []string {
	var files []string
	listFilesImpl(rulesDir, func(name string) bool { return path.Ext(name) == ".go" }, &files)
	return files
}

func ListBuildFiles() []string {
	var files []string
	listFilesImpl(embeddedFsDir, func(name string) bool { return name == "BUILD.go" }, &files)
	return files
}

func ReadFile(file string) []uint8 {
	arr, err := rules.ReadFile(path.Join(embeddedFsDir, file))
	if err != nil {
		log.Fatal("Unable to read embedded file \"", file, "\": ", err)
	}
	return arr
}
