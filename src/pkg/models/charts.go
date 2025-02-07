// pkg/models/chart.go
package models

// ChartMetadata représente la structure commune utilisée dans toute l'application
type ChartMetadata struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	Description  string `yaml:"description"`
	ApiVersion   string `yaml:"apiVersion"`
	Type         string `yaml:"type,omitempty"`
	AppVersion   string `yaml:"appVersion,omitempty"`
	Dependencies []struct {
		Name       string `yaml:"name"`
		Version    string `yaml:"version"`
		Repository string `yaml:"repository"`
	} `yaml:"dependencies,omitempty"`
}
