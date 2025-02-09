// pkg/storage/paths.go
package storage

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
		"charts", // Pour les charts
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
	reference = reference + ".json"
	return filepath.Join(pm.baseStoragePath, "manifests", name, reference)
}

func (pm *PathManager) GetChartPath(name string, version string) string {
	fileName := fmt.Sprintf("%s-%s.tgz", name, version)
	chartPath := filepath.Join(pm.GetGlobalPath(), fileName)
	logrus.Infof("Chart path: %s", chartPath)
	return chartPath
}

func (pm *PathManager) FindManifestByDigest(chartName string, digest string) string {
	// 🔍 Remove sha256: prefix if present
	digest = strings.TrimPrefix(digest, "sha256:")

	// 📂 Root charts directory
	chartsDir := filepath.Join(pm.GetGlobalPath(), chartName)

	// Lire les manifests de ce chart
	manifests, err := os.ReadDir(chartsDir)
	if err != nil {
		return ""
	}

	// Pour chaque manifest
	for _, manifest := range manifests {
		// Skip non-json files
		if !strings.HasSuffix(manifest.Name(), ".json") {
			continue
		}

		// Lire et calculer le digest
		manifestPath := filepath.Join(chartsDir, manifest.Name())
		content, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		currentDigest := fmt.Sprintf("%x", sha256.Sum256(content))

		// Si on trouve le bon digest
		if currentDigest == digest {
			return manifestPath
		}
	}

	return ""
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
