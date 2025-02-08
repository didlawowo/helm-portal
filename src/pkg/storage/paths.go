// pkg/storage/paths.go
package storage

import (
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type PathManager struct {
	baseStoragePath string
}

func NewPathManager(basePath string) *PathManager {
	return &PathManager{
		baseStoragePath: basePath,
	}
}

func (pm *PathManager) GetBlobPath(digest string) string {
	return filepath.Join(pm.baseStoragePath, "blobs", digest)
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
