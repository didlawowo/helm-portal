package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	Provider string `yaml:"provider"` // "aws", "gcp", ou "azure"
	Enabled  bool   `yaml:"enabled"`
	GCP      struct {
		Bucket    string `yaml:"bucket"`
		ProjectID string `yaml:"projectID"`
	} `yaml:"gcp"`
	AWS struct {
		Bucket string `yaml:"bucket"`
		Region string `yaml:"region"`
	} `yaml:"aws"`
	Azure struct {
		StorageAccount string `yaml:"storageAccount"`
		Container      string `yaml:"container"`
	} `yaml:"azure"`
}

type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`

	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
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

	// Azure credentials
	AzureStorageAccountKey string
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
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		config.Logging.Format = logFormat
	}

	// Paramètres de backup
	if provider := os.Getenv("BACKUP_PROVIDER"); provider != "" {
		config.Backup.Provider = provider
	}
	if enabled := os.Getenv("BACKUP_ENABLED"); enabled != "" {
		config.Backup.Enabled = enabled == "true"
	}

	// GCP backup config
	if gcpBucket := os.Getenv("GCP_BUCKET"); gcpBucket != "" {
		config.Backup.GCP.Bucket = gcpBucket
	}
	if gcpProjectID := os.Getenv("GCP_PROJECT_ID"); gcpProjectID != "" {
		config.Backup.GCP.ProjectID = gcpProjectID
	}

	// AWS backup config
	if awsBucket := os.Getenv("AWS_BUCKET"); awsBucket != "" {
		config.Backup.AWS.Bucket = awsBucket
	}
	if awsRegion := os.Getenv("AWS_REGION"); awsRegion != "" {
		config.Backup.AWS.Region = awsRegion
	}

	// Azure backup config
	if azureAccount := os.Getenv("AZURE_STORAGE_ACCOUNT"); azureAccount != "" {
		config.Backup.Azure.StorageAccount = azureAccount
	}
	if azureContainer := os.Getenv("AZURE_CONTAINER"); azureContainer != "" {
		config.Backup.Azure.Container = azureContainer
	}

	// Load auth users from environment variables
	loadAuthFromEnv(config)
}

// loadAuthFromEnv charge les utilisateurs depuis les variables d'environnement
func loadAuthFromEnv(config *Config) {
	// Option 1: Support pour utilisateurs multiples via HELM_USERS (format: "user1:pass1,user2:pass2")
	if usersEnv := os.Getenv("HELM_USERS"); usersEnv != "" {
		fmt.Printf("🔐 Loading users from HELM_USERS: %s\n", usersEnv)
		config.Auth.Users = []User{}
		for _, userPair := range strings.Split(usersEnv, ",") {
			parts := strings.SplitN(strings.TrimSpace(userPair), ":", 2)
			if len(parts) == 2 {
				user := User{
					Username: strings.TrimSpace(parts[0]),
					Password: strings.TrimSpace(parts[1]),
				}
				config.Auth.Users = append(config.Auth.Users, user)
				fmt.Printf("🔐 Added user: %s\n", user.Username)
			}
		}
		fmt.Printf("🔐 Total users loaded: %d\n", len(config.Auth.Users))
		return // Si HELM_USERS est défini, on utilise seulement ça
	}

	// Option 2: Variables d'environnement préfixées (HELM_USER_1_USERNAME, HELM_USER_1_PASSWORD, etc.)
	loadUsersFromPrefixedEnv(config)

	// Option 3: Support pour un seul utilisateur via HELM_USERNAME/HELM_PASSWORD (fallback)
	if username := os.Getenv("HELM_USERNAME"); username != "" {
		if password := os.Getenv("HELM_PASSWORD"); password != "" {
			// Si pas d'utilisateurs définis, créer un utilisateur unique
			if len(config.Auth.Users) == 0 {
				config.Auth.Users = []User{{
					Username: username,
					Password: password,
				}}
			}
		}
	}
}

// loadUsersFromPrefixedEnv charge les utilisateurs depuis des variables préfixées
// Format: HELM_USER_1_USERNAME, HELM_USER_1_PASSWORD, HELM_USER_2_USERNAME, etc.
func loadUsersFromPrefixedEnv(config *Config) {
	config.Auth.Users = []User{}
	
	// Parcourir jusqu'à 20 utilisateurs possibles (peut être ajusté)
	for i := 1; i <= 20; i++ {
		usernameKey := fmt.Sprintf("HELM_USER_%d_USERNAME", i)
		passwordKey := fmt.Sprintf("HELM_USER_%d_PASSWORD", i)
		
		username := os.Getenv(usernameKey)
		password := os.Getenv(passwordKey)
		
		if username != "" && password != "" {
			config.Auth.Users = append(config.Auth.Users, User{
				Username: username,
				Password: password,
			})
		}
	}
}

// LoadSecrets charge les secrets depuis les variables d'environnement
func LoadSecrets() *Secrets {
	secrets := &Secrets{}

	// AWS secrets
	secrets.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	secrets.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")

	// GCP secrets
	secrets.GCPCredentialsFile = os.Getenv("GCP_CREDENTIALS_FILE")

	// Azure secrets
	secrets.AzureStorageAccountKey = os.Getenv("AZURE_STORAGE_ACCOUNT_KEY")

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
