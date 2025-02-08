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

type ChartGroup struct {
	Name     string          // Nom du chart
	Versions []ChartMetadata // Liste des versions disponibles
}

func GroupChartsByName(charts []ChartMetadata) []ChartGroup {
	// Map pour regrouper les versions par nom
	chartGroups := make(map[string][]ChartMetadata)

	// Regrouper toutes les versions par nom
	for _, chart := range charts {
		chartGroups[chart.Name] = append(chartGroups[chart.Name], chart)
	}

	// Convertir la map en slice
	result := make([]ChartGroup, 0, len(chartGroups))
	for name, versions := range chartGroups {
		result = append(result, ChartGroup{
			Name:     name,
			Versions: versions,
		})
	}

	return result
}
