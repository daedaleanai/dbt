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

	"github.com/daedaleanai/dbt/v2/config"
	"github.com/daedaleanai/dbt/v2/log"
	"github.com/daedaleanai/dbt/v2/netrc"
	"github.com/daedaleanai/dbt/v2/util"
)

const tarMetadataFileName = ".metadata"

const defaultDirMode = 0770

type metadataFile struct {
	URL    string
	Sha256 string
}

// TarModule is a module backed by a tar.gz archive.
// TarModules only have a single "master" version.
type TarModule struct {
	path   string
	mirror *TarMirror
}

type TarMirror struct {
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

// Obtains a mirror for a tar module if the global mirror directory has been set up
func getOrCreateTarMirror(url string) (*TarMirror, error) {
	configuration := config.GetConfig()
	if configuration.Mirror == "" {
		log.Debug("Mirrors are not configured.\n")
		return nil, nil
	}

	urlHash := sha256.Sum256([]byte(url))
	urlHashString := fmt.Sprintf("tar-%x", urlHash[:])
	mirrorPath := path.Join(configuration.Mirror, urlHashString)

	log.Debug("Looking for mirror of '%s' in directory '%s'.\n", url, mirrorPath)

	if util.DirExists(mirrorPath) {
		log.Debug("Mirror found at '%s'.\n", mirrorPath)
		return &TarMirror{path: mirrorPath}, nil
	}

	util.MkdirAll(mirrorPath)
	mod := TarModule{mirrorPath, nil}
	if err := mod.download(url); err != nil {
		// If downloading fails, we remove the mirror path to leave a clean tree so that the
		// operation can be retried.
		util.RemoveDir(mod.path)
		return nil, err
	}
	log.Debug("Mirror downloaded at '%s'.\n", mirrorPath)

	return &TarMirror{path: mirrorPath}, nil
}

// createTarModule creates a new TarModule in the given `modulePath` by downloading
// and extracting the TAR archive reference by `url`. The origin of the module
// (i.e., the download url) is stored in a ".metadata" file inside the module directory.
func createTarModule(modulePath, url string) (Module, error) {
	mirror, err := getOrCreateTarMirror(url)
	if err != nil {
		return nil, err
	}

	module := TarModule{path: modulePath, mirror: mirror}
	err = module.clone(url)
	if err != nil {
		return nil, err
	}

	return module, err
}

func (m TarModule) Name() string {
	return path.Base(m.RootPath())
}

func (m TarModule) RootPath() string {
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
func (m TarModule) RevParse(rev string) string {
	return m.Head()
}

// IsDirty returns whether the module has any uncommited changes.
// TarModules never have any uncommited changes by definition.
func (m TarModule) IsDirty() bool {
	return false
}

func (m TarModule) IsAncestor(ancestor, rev string) bool {
	return true
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

func (m TarModule) Type() ModuleType {
	return TarGzModuleType
}

// clones a tar from either a mirror (if the tar module contains one and is valid) or downloaded from
// the network
func (m TarModule) clone(url string) error {
	// Check if it is available already in the mirror
	if m.mirror != nil {
		// Validate the mirror by making sure the metadata path is present
		metdata_path := path.Join(m.mirror.path, tarMetadataFileName)
		if util.FileExists(metdata_path) {
			err := util.CopyDirRecursively(m.mirror.path, m.path)
			return err
		}
	}

	// Mirror not available download instead
	return m.download(url)
}

// Downloads a tar.gz gziped archive from the provided url
func (m TarModule) download(url string) error {
	log.Log("Downloading '%s'.\n", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to construct HTTP request to download archive: %s", err)
	}

	if auth := netrc.GetAuthForUrl(url); auth != nil {
		log.Debug("Using netrc auth for url %q\n", url)
		request.SetBasicAuth(auth.User, auth.Password)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to download archive: %s", err)
	}
	defer response.Body.Close()

	hasher := sha256.New()
	gzFile := io.TeeReader(response.Body, hasher)

	tarFile, err := gzip.NewReader(gzFile)
	if err != nil {
		return fmt.Errorf("failed to decompress: %s", err)
	}

	tarReader := tar.NewReader(tarFile)
	tarRootDir := ""
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to decompress: %s", err)
		}

		headerRootDir := getRoot(header.Name)
		if header.Typeflag != tar.TypeDir && headerRootDir == header.Name {
			return fmt.Errorf("failed to decompress: archive can't have files outside root directory")
		}
		if tarRootDir == "" {
			tarRootDir = headerRootDir
		} else if tarRootDir != headerRootDir {
			return fmt.Errorf("failed to decompress: archive can't have more than one root directory")
		}

		// We can't assume that tarReader visits a dir before the files inside it, although this is true most of the time.
		// So if we find a file whose dir hasn't been created yet, we make it, with a sensible default access mode
		// When we eventually visit it, we set the correct mode
		switch header.Typeflag {
		case tar.TypeDir:
			dirPath := path.Join(m.path, stripRoot(header.Name))
			log.Debug("Creating directory '%s'.\n", dirPath)
			if err := os.MkdirAll(dirPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %s", err)
			}
			// We need this again because if the dir already existed os.MkdirAll does nothing
			if err := os.Chmod(dirPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to change filemode: %s", err)
			}
		case tar.TypeReg:
			filePath := path.Join(m.path, stripRoot(header.Name))
			if err := os.MkdirAll(path.Dir(filePath), defaultDirMode); err != nil {
				return fmt.Errorf("failed to create directory: %s", err)
			}
			log.Debug("Creating file '%s'.\n", filePath)
			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file: %s", err)
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return fmt.Errorf("failed to write file: %s", err)
			}
			if err := os.Chmod(filePath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to change filemode: %s", err)
			}
		case tar.TypeLink:
			if getRoot(header.Linkname) != tarRootDir {
				return fmt.Errorf("failed to decompress: archive can't have more than one root directory")
			}
			oldname := path.Join(m.path, stripRoot(header.Linkname))
			newname := path.Join(m.path, stripRoot(header.Name))
			if err := os.MkdirAll(path.Dir(newname), defaultDirMode); err != nil {
				return fmt.Errorf("failed to create directory: %s", err)
			}
			log.Debug("Creating link from '%s' to '%s'.\n", newname, oldname)
			if err = os.Link(oldname, newname); err != nil {
				return fmt.Errorf("failed to create link: %s", err)
			}
		case tar.TypeSymlink:
			newname := path.Join(m.path, stripRoot(header.Name))
			if err := os.MkdirAll(path.Dir(newname), defaultDirMode); err != nil {
				return fmt.Errorf("failed to create directory: %s", err)
			}
			log.Debug("Creating symlink from '%s' to '%s'.\n", newname, header.Linkname)
			if err := os.Symlink(header.Linkname, newname); err != nil {
				return fmt.Errorf("failed to create symlink: %s", err)
			}

		default:
			return fmt.Errorf("unknown tar type flag %d for entry '%s'", header.Typeflag, header.Name)
		}
	}

	metadata := metadataFile{url, hex.EncodeToString(hasher.Sum(nil))}
	util.WriteYaml(path.Join(m.path, tarMetadataFileName), metadata)
	return nil
}
