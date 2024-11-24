package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for our application
type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
		URL  string `yaml:"url"`
	} `yaml:"server"`

	Database struct {
		Host              string `yaml:"host"`
		Port              int    `yaml:"port"`
		Name              string `yaml:"name"`
		User              string `yaml:"user"`
		Password          string `yaml:"password"`
		SSLMode           string `yaml:"ssl_mode"`
		MaxConnections    int    `yaml:"max_connections"`
		ConnectionTimeout int    `yaml:"connection_timeout"`
	} `yaml:"database"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

// Load reads configuration from file and environment variables
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// First unmarshal into a map to handle type conversions
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw config: %w", err)
	}

	// Convert the map to YAML again
	data, err = yaml.Marshal(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw config: %w", err)
	}

	// Expand environment variables
	expandedData := os.ExpandEnv(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
