package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  port: 8080
  host: "0.0.0.0"

database:
  host: "localhost"
  port: 5432
  name: "testdb"
  user: "testuser"
  password: "testpass"
  ssl_mode: "disable"
  max_connections: 10
  connection_timeout: 5

logging:
  level: "debug"
  format: "json"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Test loading configuration
	config, err := Load(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify loaded values
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, "testdb", config.Database.Name)
	assert.Equal(t, "debug", config.Logging.Level)
}

func TestLoadWithEnvOverride(t *testing.T) {
	// Set environment variables
	t.Setenv("APP_DATABASE_HOST", "envhost")
	t.Setenv("APP_DATABASE_PORT", "5433")

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database:
  host: $APP_DATABASE_HOST
  port: $APP_DATABASE_PORT
  name: "testdb"
  user: "testuser"
  password: "testpass"
  ssl_mode: "disable"
  max_connections: 10
  connection_timeout: 5
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// Test loading configuration
	config, err := Load(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify environment variables override config file
	assert.Equal(t, "envhost", config.Database.Host)
	assert.Equal(t, 5433, config.Database.Port)
}
