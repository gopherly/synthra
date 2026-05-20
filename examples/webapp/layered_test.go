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

func TestLayeredYAMLAndEnvironmentVariables(t *testing.T) {
	t.Setenv("WEBAPP_SERVER_PORT", "9090")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_HOST", "test-db")
	t.Setenv("WEBAPP_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("WEBAPP_FEATURES_DEBUG_MODE", "false")

	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithEnv("WEBAPP_"),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	synthratest.AssertString(t, cfg, "server.host", "localhost")
	synthratest.AssertInt(t, cfg, "server.port", 9090)
	synthratest.AssertString(t, cfg, "database.primary.host", "test-db")
	synthratest.AssertInt(t, cfg, "database.primary.port", 5432)
	synthratest.AssertString(t, cfg, "auth.jwt.secret", "test-secret")
	synthratest.AssertBool(t, cfg, "features.debug.mode", false)

	var wc WebAppConfig
	cfgWithBinding, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&wc),
	)
	require.NoError(t, err)

	err = cfgWithBinding.Load(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "localhost", wc.Server.Host)
	assert.Equal(t, 9090, wc.Server.Port)
	assert.Equal(t, "test-db", wc.Database.Primary.Host)
	assert.Equal(t, 5432, wc.Database.Primary.Port)
	assert.Equal(t, "test-secret", wc.Auth.JWT.Secret)
	assert.False(t, wc.Features.Debug.Mode)
}

func TestYAMLOnlyConfiguration(t *testing.T) {
	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	synthratest.AssertString(t, cfg, "server.host", "localhost")
	synthratest.AssertInt(t, cfg, "server.port", 3000)
	synthratest.AssertString(t, cfg, "database.primary.host", "localhost")
	synthratest.AssertInt(t, cfg, "database.primary.port", 5432)
	synthratest.AssertString(t, cfg, "auth.jwt.secret", "dev-jwt-secret-change-in-production")
	synthratest.AssertBool(t, cfg, "features.debug.mode", true)
}

func TestEnvironmentVariablesOnly(t *testing.T) {
	t.Setenv("WEBAPP_SERVER_HOST", "env-host")
	t.Setenv("WEBAPP_SERVER_PORT", "8080")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_HOST", "env-db")
	t.Setenv("WEBAPP_AUTH_JWT_SECRET", "env-secret")

	cfg, err := synthra.New(
		synthra.WithEnv("WEBAPP_"),
	)
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	synthratest.AssertString(t, cfg, "server.host", "env-host")
	synthratest.AssertInt(t, cfg, "server.port", 8080)
	synthratest.AssertString(t, cfg, "database.primary.host", "env-db")
	synthratest.AssertString(t, cfg, "auth.jwt.secret", "env-secret")
}
