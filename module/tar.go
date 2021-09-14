package module

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

// TarDefaultVersion is the default version for all TarModules.
const TarDefaultVersion = "master"

const tarMetadataFileName = ".metadata"

const defaultDirMode = 0770

type metadataFile struct {
	URL    string
	Sha256 string
}

// TarModule is a module backed by a tar.gz archive.
// TarModules only have a single "master" version.
type TarModule struct {
	path string
}

func getRoot(p string) string {
	firstSlash := strings.IndexByte(p, '/')
	if firstSlash == -1 {
		return p
	}
	return p[0:firstSlash]
}

// This leaves a leading /, but this is fine because the result paths are relative to modulePath
func stripRoot(p string) string {
	root := getRoot(p)
	if p == root {
		return "/"
	}
	return p[len(root):]
}

// createTarModule creates a new TarModule in the given `modulePath` by downloading
// and extracting the TAR archive reference by `url`. The origin of the module
// (i.e., the download url) is stored in a ".metadata" file inside the module directory.
func createTarModule(modulePath, url string) (Module, error) {
	log.Log("Downloading '%s'.\n", url)

	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download archive: %s", err)
	}
	defer response.Body.Close()

	hasher := sha256.New()
	gzFile := io.TeeReader(response.Body, hasher)

	tarFile, err := gzip.NewReader(gzFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %s", err)
	}

	tarReader := tar.NewReader(tarFile)
	tarRootDir := ""
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decompress: %s", err)
		}

		headerRootDir := getRoot(header.Name)
		if header.Typeflag != tar.TypeDir && headerRootDir == header.Name {
			return nil, fmt.Errorf("failed to decompress: archive can't have files outside root directory")
		}
		if tarRootDir == "" {
			tarRootDir = headerRootDir
		} else if tarRootDir != headerRootDir {
			return nil, fmt.Errorf("failed to decompress: archive can't have more than one root directory")
		}

		// We can't assume that tarReader visits a dir before the files inside it, although this is true most of the time.
		// So if we find a file whose dir hasn't been created yet, we make it, with a sensible default access mode
		// When we eventually visit it, we set the correct mode
		switch header.Typeflag {
		case tar.TypeDir:
			dirPath := path.Join(modulePath, stripRoot(header.Name))
			log.Debug("Creating directory '%s'.\n", dirPath)
			if err := os.MkdirAll(dirPath, os.FileMode(header.Mode)); err != nil {
				return nil, fmt.Errorf("failed to create directory: %s", err)
			}
			// We need this again because if the dir already existed os.MkdirAll does nothing
			if err := os.Chmod(dirPath, os.FileMode(header.Mode)); err != nil {
				return nil, fmt.Errorf("failed to change filemode: %s", err)
			}
		case tar.TypeReg:
			filePath := path.Join(modulePath, stripRoot(header.Name))
			if err := os.MkdirAll(path.Dir(filePath), defaultDirMode); err != nil {
				return nil, fmt.Errorf("failed to create directory: %s", err)
			}
			log.Debug("Creating file '%s'.\n", filePath)
			file, err := os.Create(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to create file: %s", err)
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to write file: %s", err)
			}
			if err := os.Chmod(filePath, os.FileMode(header.Mode)); err != nil {
				return nil, fmt.Errorf("failed to change filemode: %s", err)
			}
		case tar.TypeLink:
			if getRoot(header.Linkname) != tarRootDir {
				return nil, fmt.Errorf("failed to decompress: archive can't have more than one root directory")
			}
			oldname := path.Join(modulePath, stripRoot(header.Linkname))
			newname := path.Join(modulePath, stripRoot(header.Name))
			if err := os.MkdirAll(path.Dir(newname), defaultDirMode); err != nil {
				return nil, fmt.Errorf("failed to create directory: %s", err)
			}
			log.Debug("Creating link from '%s' to '%s'.\n", newname, oldname)
			if err = os.Link(oldname, newname); err != nil {
				return nil, fmt.Errorf("failed to create link: %s", err)
			}
		case tar.TypeSymlink:
			newname := path.Join(modulePath, stripRoot(header.Name))
			if err := os.MkdirAll(path.Dir(newname), defaultDirMode); err != nil {
				return nil, fmt.Errorf("failed to create directory: %s", err)
			}
			log.Debug("Creating symlink from '%s' to '%s'.\n", newname, header.Linkname)
			if err := os.Symlink(header.Linkname, newname); err != nil {
				return nil, fmt.Errorf("failed to create symlink: %s", err)
			}

		default:
			return nil, fmt.Errorf("unknown tar type flag %d for entry '%s'", header.Typeflag, header.Name)
		}
	}

	metadata := metadataFile{url, hex.EncodeToString(hasher.Sum(nil))}
	util.WriteYaml(path.Join(modulePath, tarMetadataFileName), metadata)
	return TarModule{modulePath}, nil
}

// Name returns the name of the module.
func (m TarModule) Name() string {
	return path.Base(m.path)
}

// Path returns the on-disk path of the module.
func (m TarModule) Path() string {
	return m.path
}

// URL returns the url of the underlying tar archive.
func (m TarModule) URL() string {
	var metadata metadataFile
	util.ReadYaml(path.Join(m.path, tarMetadataFileName), &metadata)
	return metadata.URL
}

// Head returns the default version for all TarModules.
func (m TarModule) Head() string {
	var metadata metadataFile
	util.ReadYaml(path.Join(m.path, tarMetadataFileName), &metadata)
	return metadata.Sha256
}

// RevParse returns the default version for all TarModules.
func (m TarModule) RevParse(ref string) string {
	if ref != TarDefaultVersion {
		log.Fatal("Failed to parse version '%s': TarModule only has '%s' version.\n", ref, TarDefaultVersion)
	}
	return m.Head()
}

// IsDirty returns whether the module has any uncommited changes.
// TarModules never have any uncommited changes by definition.
func (m TarModule) IsDirty() bool {
	return false
}

// Fetch does nothing on TarModules and reports that no changes have been fetched.
func (m TarModule) Fetch() bool {
	return false
}

// Checkout changes the module's current version to `ref`.
// TarModules only have a single version. Attempting to check out any
// other version results in an error.
func (m TarModule) Checkout(hash string) {
	if hash != m.Head() {
		log.Fatal("Failed to checkout version '%s': cannot change version of TarModule.\n", hash)
	}
}
