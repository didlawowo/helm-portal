package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// pkg/config/config.go
type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type AuthConfig struct {
	Enabled bool   `yaml:"enabled"`
	Users   []User `yaml:"users"`
}

type Backup struct {
	GCP struct {
		Bucket          string `yaml:"bucket"`
		ProjectID       string `yaml:"projectID"`
		CredentialsFile string `yaml:"credentialsFile"`
	} `yaml:"gcp"`
	AWS struct {
		Bucket          string `yaml:"bucket"`
		Region          string `yaml:"region"`
		AccessKeyID     string `yaml:"accessKeyID"`
		SecretAccessKey string `yaml:"secretAccessKey"`
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

func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("❌ error reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("❌ error parsing config: %w", err)
	}

	return config, nil
}
