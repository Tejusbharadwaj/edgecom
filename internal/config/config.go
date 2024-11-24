package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for our application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
	URL  string `mapstructure:"url"`
}

type DatabaseConfig struct {
	Host              string `mapstructure:"host"`
	Port              int    `mapstructure:"port"`
	Name              string `mapstructure:"name"`
	User              string `mapstructure:"user"`
	Password          string `mapstructure:"password"`
	SSLMode           string `mapstructure:"ssl_mode"`
	MaxConnections    int    `mapstructure:"max_connections"`
	ConnectionTimeout int    `mapstructure:"connection_timeout"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
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

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")

	v.SetDefault("database.port", 5432)
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_connections", 10)
	v.SetDefault("database.connection_timeout", 5)

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}
