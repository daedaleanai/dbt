package module

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

// TarDefaultVersion is the default version for all TarModules.
const TarDefaultVersion = "master"

const tarMetadataFileName = ".metadata"

type metadataFile struct {
	URL    string
	Sha256 string
}

// TarModule is a module backed by a tar.gz archive.
// TarModules only have a single "master" version.
type TarModule struct {
	path string
}

// createTarModule creates a new TarModule in the given `modulePath` by downloading
// and extracting the TAR archive reference by `url`. The origin of the module
// (i.e., the download url) is stored in a ".metadata" file inside the module directory.
func createTarModule(modulePath, url string) Module {
	log.Log("Downloading '%s'.\n", url)
	log.Spinner.Start()
	defer log.Spinner.Stop()

	response, err := http.Get(url)
	if err != nil {
		log.Fatal("Failed to download archive: %s.\n", err)
	}
	defer response.Body.Close()

	hasher := sha256.New()
	gzFile := io.TeeReader(response.Body, hasher)

	tarFile, err := gzip.NewReader(gzFile)
	if err != nil {
		log.Fatal("Failed to decompress: %s.\n", err)
	}

	tarReader := tar.NewReader(tarFile)
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("Failed to decompress: %s.\n", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dirPath := path.Join(modulePath, header.Name)
			log.Debug("Creating directory '%s'.\n", dirPath)
			err := os.MkdirAll(dirPath, os.FileMode(header.Mode))
			if err != nil {
				log.Fatal("Failed to create directory while decompressing archive: %s.\n", err)
			}
		case tar.TypeReg:
			filePath := path.Join(modulePath, header.Name)
			log.Debug("Creating file '%s'.\n", filePath)
			file, err := os.Create(filePath)
			if err != nil {
				log.Fatal("Failed to create file while decompressing archive: %s.\n", err)
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				log.Fatal("Failed to writing file while decompressing archive: %s.\n", err)
			}
			err = os.Chmod(filePath, os.FileMode(header.Mode))
			if err != nil {
				log.Fatal("Failed to change filemode while decompressing archive: %s.\n", err)
			}
		case tar.TypeLink:
			oldname := path.Join(modulePath, header.Linkname)
			newname := path.Join(modulePath, header.Name)
			log.Debug("Creating link from '%s' to '%s'.\n", newname, oldname)
			err = os.Link(oldname, newname)
			if err != nil {
				log.Fatal("Failed to create link while decompressing archive: %s.\n", err)
			}
		case tar.TypeSymlink:
			newname := path.Join(modulePath, header.Name)
			log.Debug("Creating symlink from '%s' to '%s'.\n", newname, header.Linkname)
			err = os.Symlink(header.Linkname, newname)
			if err != nil {
				log.Fatal("Failed to create symlink while decompressing archive: %s.\n", err)
			}

		default:
			log.Fatal("Failed to decompress archive: unknown tar type flag %d for entry '%s'.\n", header.Typeflag, header.Name)
		}
	}

	metadata := metadataFile{url, hex.EncodeToString(hasher.Sum(nil))}
	util.WriteYaml(path.Join(modulePath, tarMetadataFileName), metadata)
	return TarModule{modulePath}
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
