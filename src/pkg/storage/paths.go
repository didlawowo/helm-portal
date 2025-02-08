// pkg/storage/paths.go
package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type PathManager struct {
	baseStoragePath string
	log             *logrus.Logger
}

func NewPathManager(basePath string) *PathManager {
	// Créer les dossiers nécessaires
	dirs := []string{
		"temp",  // Pour les uploads temporaires
		"blobs", // Pour les blobs
		"manifests",
	}

	for _, dir := range dirs {
		path := filepath.Join(basePath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", path, err)
		}
	}

	return &PathManager{

		baseStoragePath: basePath,
	}
}

// Dans storage/path_manager.go

func (pm *PathManager) GetTempPath(uuid string) string {
	return filepath.Join(pm.baseStoragePath, "temp", uuid)
}

func (pm *PathManager) GetBlobPath(digest string) string {
	return filepath.Join(pm.baseStoragePath, "blobs", digest)
}

func (pm *PathManager) GetManifestPath(name, reference string) string {
	return filepath.Join(pm.baseStoragePath, "manifests", name, reference)
}

func (pm *PathManager) GetChartPath(name string, version string) string {
	fileName := fmt.Sprintf("%s-%s.tgz", name, version)
	chartPath := filepath.Join(pm.GetGlobalPath(), fileName)
	logrus.Infof("Chart path: %s", chartPath)
	return chartPath
}
func (pm *PathManager) GetGlobalPath() string {
	return filepath.Join(pm.baseStoragePath, "charts")
}

func (pm *PathManager) GetOCIRepositoryPath(name string) string {
	return filepath.Join(pm.baseStoragePath, "oci", name)
}

func (pm *PathManager) GetIndexPath() string {
	return filepath.Join(pm.baseStoragePath, "index.yaml")
}
