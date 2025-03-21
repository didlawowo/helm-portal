package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

// pkg/config/config.go
type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type AuthConfig struct {
	Users []User `yaml:"users"`
}

type Backup struct {
	Provider string `yaml:"provider"` // "aws" ou "gcp"
	Enabled  bool   `yaml:"enabled"`
	GCP      struct {
		Bucket    string `yaml:"bucket"`
		ProjectID string `yaml:"projectID"`
	} `yaml:"gcp"`
	AWS struct {
		Bucket string `yaml:"bucket"`
		Region string `yaml:"region"`
	} `yaml:"aws"`
}

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`

	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
	Auth   AuthConfig `yaml:"auth"`
	Backup Backup     `yaml:"backup"`
}

type Secrets struct {
	// AWS credentials
	AWSAccessKeyID     string
	AWSSecretAccessKey string

	// GCP credentials
	GCPCredentialsFile string
}

// LoadConfig charge la configuration depuis un fichier YAML
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("❌ error reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("❌ error parsing config: %w", err)
	}

	// Charger les configs depuis des variables d'environnement si présentes
	loadConfigFromEnv(config)

	fmt.Printf("🔍 Loaded config successfully\n")
	return config, nil
}

// Charge les configurations depuis les variables d'environnement
func loadConfigFromEnv(config *Config) {
	// Paramètres du serveur
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Server.Port = port
		}
	}

	// Paramètres de stockage
	// if storagePath := os.Getenv("STORAGE_PATH"); storagePath != "" {
	// 	config.Storage.Path = storagePath
	// }

	// Paramètres de logging
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}

	// Paramètres de backup (sauf secrets)
	// if provider := os.Getenv("BACKUP_PROVIDER"); provider != "" {
	// 	config.Backup.Provider = provider
	// }

	// if gcpBucket := os.Getenv("GCP_BUCKET"); gcpBucket != "" {
	// 	config.Backup.GCP.Bucket = gcpBucket
	// }

	// if gcpProjectID := os.Getenv("GCP_PROJECT_ID"); gcpProjectID != "" {
	// 	config.Backup.GCP.ProjectID = gcpProjectID
	// }

	// if awsBucket := os.Getenv("AWS_BUCKET"); awsBucket != "" {
	// 	config.Backup.AWS.Bucket = awsBucket
	// }

	// if awsRegion := os.Getenv("AWS_REGION"); awsRegion != "" {
	// 	config.Backup.AWS.Region = awsRegion
	// }

}

// LoadSecrets charge les secrets depuis les variables d'environnement
func LoadSecrets() *Secrets {
	secrets := &Secrets{}

	// AWS secrets
	secrets.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	secrets.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")

	// GCP secrets
	secrets.GCPCredentialsFile = os.Getenv("GCP_CREDENTIALS_FILE")

	return secrets
}

// LoadAuthFromFile charge les informations d'authentification depuis un fichier séparé
func LoadAuthFromFile(config *Config) error {
	// Chercher le fichier d'authentification
	credFile := os.Getenv("AUTH_FILE")
	if credFile == "" {
		// Utiliser un emplacement par défaut si la variable d'environnement n'est pas définie
		credFile = "config/auth.yaml"
	}

	// Vérifier si le fichier existe
	if _, err := os.Stat(credFile); os.IsNotExist(err) {
		return fmt.Errorf("auth file %s does not exist, using default auth config", credFile)
	}

	// Lire et parser le fichier d'authentification
	data, err := os.ReadFile(credFile)
	if err != nil {
		return fmt.Errorf("error reading auth file: %w", err)
	}

	// Structure temporaire pour le chargement
	var authConfig struct {
		Auth AuthConfig `yaml:"auth"`
	}

	if err := yaml.Unmarshal(data, &authConfig); err != nil {
		return fmt.Errorf("error parsing auth file: %w", err)
	}

	// Mettre à jour la configuration avec les données d'authentification
	config.Auth = authConfig.Auth

	return nil
}
