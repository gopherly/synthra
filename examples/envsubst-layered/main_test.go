// Copyright 2026 The Gopherly Authors
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

//go:build !integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
)

// loadCfg is the same resolver composition as main(), extracted for tests so
// we can vary the .env content without touching the real filesystem.
func loadCfg(t *testing.T, envFileContent string, prefixedEnv map[string]string) *synthra.Synthra {
	t.Helper()

	// Write a temp .env file.
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte(envFileContent), 0o600))

	envFile, err := synthra.FromEnvFile(envPath)
	require.NoError(t, err)

	// Set DPY_VAR_* variables in the test environment.
	for k, v := range prefixedEnv {
		t.Setenv("DPY_VAR_"+k, v)
	}

	// Use the same config.yaml as the example.
	cfg, err := synthra.New(
		synthra.WithFileFS(os.DirFS("."), "config.yaml"),
		synthra.WithEnvSubst(
			synthra.FromEnv().Prefix("DPY_VAR_").
				Or(envFile).
				Or(synthra.FromMap(manifestDefaults)),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	return cfg
}

func TestLayered_StaticDefaultsUsedWhenNothingElseSet(t *testing.T) {
	t.Parallel()

	cfg := loadCfg(t, "", nil)

	region, err := cfg.String("region")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", region) // from manifestDefaults

	image, err := cfg.String("image")
	require.NoError(t, err)
	assert.Equal(t, "my-app:stable", image) // TAG=stable from manifestDefaults

	dbURL, err := cfg.String("db_url")
	require.NoError(t, err)
	assert.Equal(t, "postgres://db.internal:5432/mydb", dbURL) // DB_HOST from manifestDefaults
}

func TestLayered_EnvFileOverridesDefaults(t *testing.T) {
	t.Parallel()

	cfg := loadCfg(t, "REGION=us-east-1\nTAG=v1.2.3\n", nil)

	region, err := cfg.String("region")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", region) // .env file wins over manifestDefaults

	image, err := cfg.String("image")
	require.NoError(t, err)
	assert.Equal(t, "my-app:v1.2.3", image) // TAG from .env file

	dbURL, err := cfg.String("db_url")
	require.NoError(t, err)
	assert.Equal(t, "postgres://db.internal:5432/mydb", dbURL) // DB_HOST from manifestDefaults (not in .env)
}

func TestLayered_PrefixedOSEnvOverridesEnvFile(t *testing.T) {
	cfg := loadCfg(t,
		"REGION=us-east-1\nTAG=v1.2.3\n",
		map[string]string{
			"REGION": "ap-southeast-1", // DPY_VAR_REGION overrides .env REGION
			"TAG":    "canary",         // DPY_VAR_TAG overrides .env TAG
		},
	)

	region, err := cfg.String("region")
	require.NoError(t, err)
	assert.Equal(t, "ap-southeast-1", region) // DPY_VAR_REGION wins

	image, err := cfg.String("image")
	require.NoError(t, err)
	assert.Equal(t, "my-app:canary", image) // DPY_VAR_TAG wins

	dbURL, err := cfg.String("db_url")
	require.NoError(t, err)
	assert.Equal(t, "postgres://db.internal:5432/mydb", dbURL) // only in manifestDefaults
}

func TestLayered_DefaultFallbackFiredWhenNobodyOwnsTheVar(t *testing.T) {
	t.Parallel()

	// None of the three resolver layers define TAG.
	// The ${TAG:-latest} fallback in config.yaml should fire.
	defaults := map[string]string{
		"REGION":  "eu-west-1",
		"DB_HOST": "db.internal",
		// TAG intentionally absent
	}

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte(""), 0o600))

	envFile, err := synthra.FromEnvFile(envPath)
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFileFS(os.DirFS("."), "config.yaml"),
		synthra.WithEnvSubst(
			synthra.FromEnv().Prefix("DPY_VAR_").
				Or(envFile).
				Or(synthra.FromMap(defaults)),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	image, err := cfg.String("image")
	require.NoError(t, err)
	assert.Equal(t, "my-app:latest", image) // ${TAG:-latest} fallback
}
