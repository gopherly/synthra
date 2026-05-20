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

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/synthratest"
)

func TestWebAppConfig_EnvironmentVariables(t *testing.T) {
	// Set up test environment variables (all required fields)
	t.Setenv("WEBAPP_SERVER_HOST", "test-host")
	t.Setenv("WEBAPP_SERVER_PORT", "9090")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_HOST", "test-db")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_PORT", "5432")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_DATABASE", "testdb")
	t.Setenv("WEBAPP_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("WEBAPP_AUTH_TOKEN_DURATION", "1h")
	t.Setenv("WEBAPP_FEATURES_DEBUG_MODE", "true")

	// Create configuration without binding to test direct access
	cfg, err := synthra.New(
		synthra.WithEnv("WEBAPP_"),
	)
	require.NoError(t, err)

	// Load configuration
	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// Test direct configuration access
	synthratest.AssertString(t, cfg, "server.host", "test-host")
	synthratest.AssertInt(t, cfg, "server.port", 9090)
	synthratest.AssertString(t, cfg, "database.primary.host", "test-db")
	synthratest.AssertInt(t, cfg, "database.primary.port", 5432)
	synthratest.AssertString(t, cfg, "database.primary.database", "testdb")
	synthratest.AssertString(t, cfg, "auth.jwt.secret", "test-secret")
	synthratest.AssertBool(t, cfg, "features.debug.mode", true)

	// Now test with binding
	var wc WebAppConfig
	cfgWithBinding, err := synthra.New(
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&wc),
	)
	require.NoError(t, err)

	// Load configuration with binding
	err = cfgWithBinding.Load(context.Background())
	require.NoError(t, err)

	// Test struct binding
	assert.Equal(t, "test-host", wc.Server.Host)
	assert.Equal(t, 9090, wc.Server.Port)
	assert.Equal(t, "test-db", wc.Database.Primary.Host)
	assert.Equal(t, 5432, wc.Database.Primary.Port)
	assert.Equal(t, "testdb", wc.Database.Primary.Database)
	assert.Equal(t, "test-secret", wc.Auth.JWT.Secret)
	assert.True(t, wc.Features.Debug.Mode)
}

func TestWebAppConfig_NestedStructures(t *testing.T) {
	// Test nested environment variable mapping (including required fields)
	t.Setenv("WEBAPP_SERVER_HOST", "test-host")
	t.Setenv("WEBAPP_SERVER_PORT", "9090")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_HOST", "test-db")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_PORT", "5432")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_DATABASE", "testdb")
	t.Setenv("WEBAPP_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("WEBAPP_AUTH_TOKEN_DURATION", "1h")
	t.Setenv("WEBAPP_SERVER_TLS_ENABLED", "true")
	t.Setenv("WEBAPP_SERVER_TLS_CERT_FILE", "/path/to/cert.pem")
	t.Setenv("WEBAPP_SERVER_TLS_KEY_FILE", "/path/to/key.pem")
	t.Setenv("WEBAPP_DATABASE_POOL_MAX_OPEN", "50")
	t.Setenv("WEBAPP_DATABASE_POOL_MAX_IDLE", "10")

	// Test direct access first
	cfg, err := synthra.New(
		synthra.WithEnv("WEBAPP_"),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	// Test direct access to nested values
	synthratest.AssertBool(t, cfg, "server.tls.enabled", true)
	synthratest.AssertString(t, cfg, "server.tls.cert.file", "/path/to/cert.pem")
	synthratest.AssertString(t, cfg, "server.tls.key.file", "/path/to/key.pem")
	synthratest.AssertInt(t, cfg, "database.pool.max.open", 50)
	synthratest.AssertInt(t, cfg, "database.pool.max.idle", 10)

	// Now test with binding
	var wc WebAppConfig
	cfgWithBinding, err := synthra.New(
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&wc),
	)
	require.NoError(t, err)

	err = cfgWithBinding.Load(context.Background())
	require.NoError(t, err)

	// Test nested TLS configuration
	assert.True(t, wc.Server.TLS.Enabled)
	assert.Equal(t, "/path/to/cert.pem", wc.Server.TLS.Cert.File)
	assert.Equal(t, "/path/to/key.pem", wc.Server.TLS.Key.File)

	// Test nested database pool configuration
	assert.Equal(t, 50, wc.Database.Pool.Max.Open)
	assert.Equal(t, 10, wc.Database.Pool.Max.Idle)
}

func TestWebAppConfig_Validate_InvalidPort(t *testing.T) {
	t.Setenv("WEBAPP_SERVER_HOST", "localhost")
	t.Setenv("WEBAPP_SERVER_PORT", "0")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_HOST", "db")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_PORT", "5432")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_DATABASE", "app")
	t.Setenv("WEBAPP_AUTH_JWT_SECRET", "secret")
	t.Setenv("WEBAPP_AUTH_TOKEN_DURATION", "1h")

	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&WebAppConfig{}),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.Error(t, err)
}

func TestWebAppConfig_Validate_TLSRequiresCertFiles(t *testing.T) {
	t.Setenv("WEBAPP_SERVER_HOST", "localhost")
	t.Setenv("WEBAPP_SERVER_PORT", "8080")
	t.Setenv("WEBAPP_SERVER_TLS_ENABLED", "true")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_HOST", "db")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_PORT", "5432")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_DATABASE", "app")
	t.Setenv("WEBAPP_AUTH_JWT_SECRET", "secret")
	t.Setenv("WEBAPP_AUTH_TOKEN_DURATION", "1h")
	// Deliberately omit cert and key paths

	cfg, err := synthra.New(
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&WebAppConfig{}),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.Error(t, err)
}
