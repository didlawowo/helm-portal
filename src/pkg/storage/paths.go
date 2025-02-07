// pkg/storage/paths.go
package storage

import (
	"path/filepath"
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

func (pm *PathManager) GetChartPath(name, version string) string {
	return filepath.Join(pm.baseStoragePath, name, version)
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
