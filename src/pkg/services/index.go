// internal/chart/service/index.go

package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"helm-portal/config"
	"os"
	"path/filepath"
	"time"
)

// IndexFile représente la structure de index.yaml
type IndexFile struct {
	APIVersion string                     `yaml:"apiVersion"`
	Generated  time.Time                  `yaml:"generated"`
	Entries    map[string][]*ChartVersion `yaml:"entries"`
}

// ChartVersion représente une version spécifique d'un chart
type ChartVersion struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	Description string    `yaml:"description"`
	AppVersion  string    `yaml:"appVersion,omitempty"`
	APIVersion  string    `yaml:"apiVersion,omitempty"`
	Created     time.Time `yaml:"created"`
	Digest      string    `yaml:"digest"` // SHA256 du fichier
	URLs        []string  `yaml:"urls"`   // URLs de téléchargement
}

type IndexService struct {
	storagePath    string
	config         *config.Config
	log            *logrus.Logger
	baseURL        string
	chartExtractor ChartExtractor
}

func NewIndexService(config *config.Config, log *logrus.Logger, chartExtractor ChartExtractor) *IndexService {
	if err := os.MkdirAll(config.Storage.Path, 0755); err != nil {
		log.WithError(err).Error("❌ Impossible de créer le dossier de stockage")
	}

	return &IndexService{
		storagePath:    config.Storage.Path,
		config:         config,
		log:            log,
		baseURL:        config.Helm.BaseURL,
		chartExtractor: chartExtractor,
	}
}

func (s *IndexService) EnsureIndexExists() error {
	indexPath := filepath.Join(s.storagePath, "index.yaml")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return s.UpdateIndex()
	}
	return nil
}

func (s *IndexService) GetIndexPath() string {
	return filepath.Join(s.storagePath, "index.yaml")
}

// generateIndex crée ou met à jour le fichier index.yaml
func (s *IndexService) UpdateIndex() error {
	s.log.Info("🔄 Génération de l'index.yaml")

	// Créer un nouvel index
	index := &IndexFile{
		APIVersion: "v1",
		Generated:  time.Now(),
		Entries:    make(map[string][]*ChartVersion),
	}

	// Lire le répertoire des charts
	chartsDir := filepath.Join(s.storagePath, "charts")
	files, err := os.ReadDir(chartsDir)
	if err != nil {
		return fmt.Errorf("❌ erreur lecture répertoire charts: %w", err)
	}

	// Traiter chaque fichier .tgz
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".tgz" {
			continue
		}

		// Lire le fichier chart
		chartPath := filepath.Join(chartsDir, file.Name())
		chartData, err := os.ReadFile(chartPath)
		if err != nil {
			s.log.WithError(err).WithField("file", file.Name()).Error("❌ Erreur lecture chart")
			continue
		}

		// Extraire les métadonnées
		metadata, err := s.chartExtractor.ExtractChartMetadata(chartData)
		if err != nil {
			s.log.WithError(err).WithField("file", file.Name()).Error("❌ Erreur extraction métadonnées")
			continue
		}

		// Calculer le digest SHA256
		digest := sha256.Sum256(chartData)
		digestStr := hex.EncodeToString(digest[:])

		// Créer l'URL de téléchargement
		downloadURL := fmt.Sprintf("%s/charts/%s", s.baseURL, file.Name())

		// Créer la version du chart
		chartVersion := &ChartVersion{
			Name:        metadata.Name,
			Version:     metadata.Version,
			Description: metadata.Description,
			AppVersion:  metadata.AppVersion,
			APIVersion:  metadata.ApiVersion,
			Created:     time.Now(),
			Digest:      digestStr,
			URLs:        []string{downloadURL},
		}

		// Ajouter à l'index
		if _, exists := index.Entries[metadata.Name]; !exists {
			index.Entries[metadata.Name] = []*ChartVersion{}
		}
		index.Entries[metadata.Name] = append(index.Entries[metadata.Name], chartVersion)

		s.log.WithFields(logrus.Fields{
			"name":    metadata.Name,
			"version": metadata.Version,
			"digest":  digestStr[:8], // Log seulement les 8 premiers caractères
		}).Debug("✅ Chart ajouté à l'index")
	}

	// Convertir en YAML
	indexYAML, err := yaml.Marshal(index)
	if err != nil {
		return fmt.Errorf("❌ erreur marshaling index: %w", err)
	}

	// Sauvegarder le fichier
	indexPath := filepath.Join(s.storagePath, "index.yaml")
	if err := os.WriteFile(indexPath, indexYAML, 0644); err != nil {
		return fmt.Errorf("❌ erreur sauvegarde index: %w", err)
	}

	s.log.Info("✅ Index.yaml généré avec succès")
	return nil
}
