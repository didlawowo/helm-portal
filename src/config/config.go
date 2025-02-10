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

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`

	Helm struct {
		BaseURL      string `yaml:"baseURL"`
		MaxChartSize string `yaml:"maxChartSize"`
	} `yaml:"helm"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
	Auth AuthConfig `yaml:"auth"`
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
