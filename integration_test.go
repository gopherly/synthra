// Copyright 2026 The Gopherly Authors
// Copyright 2025 Company.info B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package synthra_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/codec"
	"gopherly.dev/synthra/synthratest"
)

// TestIntegration_FileSourceWithYAML tests end-to-end YAML file loading.
func TestIntegration_FileSourceWithYAML(t *testing.T) {
	t.Parallel()

	// Create temporary YAML file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	yamlContent := []byte(`
server:
  host: localhost
  port: 8080
  tls:
    enabled: true
    cert: /path/to/cert.pem
database:
  driver: postgres
  host: db.example.com
  port: 5432
  credentials:
    user: dbuser
    password: dbpass
logging:
  level: info
  format: json
`)

	err := os.WriteFile(configFile, yamlContent, 0o600)
	require.NoError(t, err)

	// Load configuration
	cfg, err := synthra.New(
		synthra.WithFileAs(configFile, codec.YAML),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// Verify loaded values
	synthratest.AssertString(t, cfg, "server.host", "localhost")
	synthratest.AssertInt(t, cfg, "server.port", 8080)
	synthratest.AssertBool(t, cfg, "server.tls.enabled", true)
	synthratest.AssertString(t, cfg, "server.tls.cert", "/path/to/cert.pem")
	synthratest.AssertString(t, cfg, "database.driver", "postgres")
	synthratest.AssertString(t, cfg, "database.host", "db.example.com")
	synthratest.AssertInt(t, cfg, "database.port", 5432)
	synthratest.AssertString(t, cfg, "database.credentials.user", "dbuser")
	synthratest.AssertString(t, cfg, "logging.level", "info")
	synthratest.AssertString(t, cfg, "logging.format", "json")
}

// TestIntegration_FileSourceWithJSON tests end-to-end JSON file loading.
func TestIntegration_FileSourceWithJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	jsonContent := []byte(`{
		"app": {
			"name": "MyApp",
			"version": "1.0.0",
			"features": ["auth", "api", "metrics"]
		},
		"cache": {
			"enabled": true,
			"ttl": 3600
		}
	}`)

	err := os.WriteFile(configFile, jsonContent, 0o600)
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFileAs(configFile, codec.JSON),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	synthratest.AssertString(t, cfg, "app.name", "MyApp")
	synthratest.AssertString(t, cfg, "app.version", "1.0.0")
	synthratest.AssertStringSlice(t, cfg, "app.features", []string{"auth", "api", "metrics"})
	synthratest.AssertBool(t, cfg, "cache.enabled", true)
	synthratest.AssertInt(t, cfg, "cache.ttl", 3600)
}

// TestIntegration_FileSourceWithTOML tests end-to-end TOML file loading.
func TestIntegration_FileSourceWithTOML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.toml")

	tomlContent := []byte(`
title = "My Application"

[server]
host = "0.0.0.0"
port = 9090

[database]
driver = "mysql"
max_connections = 100
`)

	err := os.WriteFile(configFile, tomlContent, 0o600)
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFileAs(configFile, codec.TOML),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	synthratest.AssertString(t, cfg, "title", "My Application")
	synthratest.AssertString(t, cfg, "server.host", "0.0.0.0")
	synthratest.AssertInt(t, cfg, "server.port", 9090)
	synthratest.AssertString(t, cfg, "database.driver", "mysql")
	synthratest.AssertInt(t, cfg, "database.max_connections", 100)
}

// TestIntegration_MultipleSources tests merging configurations from multiple sources.
func TestIntegration_MultipleSources(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Base configuration (defaults)
	baseFile := filepath.Join(tmpDir, "base.yaml")
	baseContent := []byte(`
server:
  host: localhost
  port: 8080
  timeout: 30
database:
  pool_size: 10
  timeout: 5
`)
	err := os.WriteFile(baseFile, baseContent, 0o600)
	require.NoError(t, err)

	// Environment-specific override
	envFile := filepath.Join(tmpDir, "production.yaml")
	envContent := []byte(`
server:
  host: 0.0.0.0
  port: 80
database:
  pool_size: 50
`)
	err = os.WriteFile(envFile, envContent, 0o600)
	require.NoError(t, err)

	// Local overrides
	localFile := filepath.Join(tmpDir, "local.yaml")
	localContent := []byte(`
server:
  port: 9090
`)
	err = os.WriteFile(localFile, localContent, 0o600)
	require.NoError(t, err)

	// Load all sources (later sources override earlier ones)
	cfg, err := synthra.New(
		synthra.WithFileAs(baseFile, codec.YAML),
		synthra.WithFileAs(envFile, codec.YAML),
		synthra.WithFileAs(localFile, codec.YAML),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// Verify merged values
	synthratest.AssertString(t, cfg, "server.host", "0.0.0.0") // from production
	synthratest.AssertInt(t, cfg, "server.port", 9090)         // from local (highest priority)
	synthratest.AssertInt(t, cfg, "server.timeout", 30)        // from base (not overridden)
	synthratest.AssertInt(t, cfg, "database.pool_size", 50)    // from production
	synthratest.AssertInt(t, cfg, "database.timeout", 5)       // from base (not overridden)
}

// TestIntegration_BindingWithValidation tests struct binding with validation.
func TestIntegration_BindingWithValidation(t *testing.T) {
	t.Parallel()

	type ServerConfig struct {
		Host string `synthra:"host"`
		Port int    `synthra:"port"`
	}

	type DatabaseConfig struct {
		Driver   string `synthra:"driver"`
		Host     string `synthra:"host"`
		Port     int    `synthra:"port"`
		Username string `synthra:"username"`
		Password string `synthra:"password"`
	}

	type AppConfig struct {
		Server   ServerConfig   `synthra:"server"`
		Database DatabaseConfig `synthra:"database"`
	}

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	yamlContent := []byte(`
server:
  host: localhost
  port: 8080
database:
  driver: postgres
  host: localhost
  port: 5432
  username: testuser
  password: testpass
`)

	err := os.WriteFile(configFile, yamlContent, 0o600)
	require.NoError(t, err)

	var appConfig AppConfig
	cfg, err := synthra.New(
		synthra.WithFileAs(configFile, codec.YAML),
		synthra.WithBinding(&appConfig),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// Verify bound struct
	assert.Equal(t, "localhost", appConfig.Server.Host)
	assert.Equal(t, 8080, appConfig.Server.Port)
	assert.Equal(t, "postgres", appConfig.Database.Driver)
	assert.Equal(t, "localhost", appConfig.Database.Host)
	assert.Equal(t, 5432, appConfig.Database.Port)
	assert.Equal(t, "testuser", appConfig.Database.Username)
	assert.Equal(t, "testpass", appConfig.Database.Password)
}

// TestIntegration_ReloadConfiguration tests reloading configuration.
func TestIntegration_ReloadConfiguration(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Initial configuration
	initialContent := []byte(`
version: 1
feature_flags:
  new_ui: false
`)

	err := os.WriteFile(configFile, initialContent, 0o600)
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFileAs(configFile, codec.YAML),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	synthratest.AssertInt(t, cfg, "version", 1)
	synthratest.AssertBool(t, cfg, "feature_flags.new_ui", false)

	// Update configuration file
	updatedContent := []byte(`
version: 2
feature_flags:
  new_ui: true
`)

	err = os.WriteFile(configFile, updatedContent, 0o600)
	require.NoError(t, err)

	// Reload configuration
	err = cfg.Load(context.Background())
	require.NoError(t, err)

	synthratest.AssertInt(t, cfg, "version", 2)
	synthratest.AssertBool(t, cfg, "feature_flags.new_ui", true)
}

// TestIntegration_FileDumper tests dumping configuration to a file.
func TestIntegration_FileDumper(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.yaml")
	dumpFile := filepath.Join(tmpDir, "dump.yaml")

	sourceContent := []byte(`
app:
  name: TestApp
  version: 1.0.0
`)

	err := os.WriteFile(sourceFile, sourceContent, 0o600)
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFileAs(sourceFile, codec.YAML),
		synthra.WithFileDumperAs(dumpFile, codec.YAML),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// Dump configuration
	err = cfg.Dump(context.Background())
	require.NoError(t, err)

	// Verify dumped file exists and contains correct data
	//nolint:gosec // Test file read is safe
	dumpedContent, err := os.ReadFile(dumpFile)
	require.NoError(t, err)
	assert.Contains(t, string(dumpedContent), "TestApp")
	assert.Contains(t, string(dumpedContent), "1.0.0")
}

// TestIntegration_CaseInsensitiveKeys tests case-insensitive key access.
func TestIntegration_CaseInsensitiveKeys(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	yamlContent := []byte(`
Server:
  Host: localhost
  Port: 8080
Database:
  Driver: postgres
`)

	err := os.WriteFile(configFile, yamlContent, 0o600)
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFileAs(configFile, codec.YAML),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// All case variations should work
	synthratest.AssertString(t, cfg, "server.host", "localhost")
	synthratest.AssertString(t, cfg, "Server.Host", "localhost")
	synthratest.AssertString(t, cfg, "SERVER.HOST", "localhost")
	synthratest.AssertInt(t, cfg, "server.port", 8080)
	synthratest.AssertInt(t, cfg, "Server.Port", 8080)
	synthratest.AssertString(t, cfg, "database.driver", "postgres")
	synthratest.AssertString(t, cfg, "DATABASE.DRIVER", "postgres")
}

// TestIntegration_EnvironmentVariables tests environment variable source.
func TestIntegration_EnvironmentVariables(t *testing.T) {
	// NOTE: Cannot use t.Parallel() with t.Setenv()

	// Set test environment variables
	t.Setenv("TESTAPP_SERVER_HOST", "envhost")
	t.Setenv("TESTAPP_SERVER_PORT", "9090")
	t.Setenv("TESTAPP_DEBUG", "true")

	cfg, err := synthra.New(
		synthra.WithEnv("TESTAPP_"),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// Environment variables should be accessible with dot notation
	synthratest.AssertString(t, cfg, "server.host", "envhost")
	synthratest.AssertString(t, cfg, "server.port", "9090")
	synthratest.AssertString(t, cfg, "debug", "true")
}
