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

package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
)

func loadConfig(t *testing.T) *synthra.Synthra {
	t.Helper()
	schema, err := os.ReadFile("schema.json")
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithJSONSchema(schema),
		synthra.WithEnvSubst(synthra.FromMap(map[string]string{"ENV": "production"})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	return cfg
}

func TestSchemaDefaults_TopLevelFields(t *testing.T) {
	cfg := loadConfig(t)

	// Explicitly set in config.yaml
	svc, err := cfg.String("service")
	require.NoError(t, err)
	assert.Equal(t, "my-app", svc)

	// Filled from schema "default"
	port, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, port)

	logLevel, err := cfg.String("log_level")
	require.NoError(t, err)
	assert.Equal(t, "info", logLevel)
}

func TestSchemaDefaults_NestedObject(t *testing.T) {
	cfg := loadConfig(t)

	// Set in config.yaml
	maxConn, err := cfg.Int("server.max_connections")
	require.NoError(t, err)
	assert.Equal(t, 50, maxConn)

	// Filled from schema "default"
	timeout, err := cfg.String("server.timeout")
	require.NoError(t, err)
	assert.Equal(t, "30s", timeout)
}

func TestSchemaDefaults_PatternProperties(t *testing.T) {
	cfg := loadConfig(t)

	// web component: only image specified; role and replicas from schema defaults
	webRole, err := cfg.String("components.web.role")
	require.NoError(t, err)
	assert.Equal(t, "service", webRole)

	webReplicas, err := cfg.Int("components.web.replicas")
	require.NoError(t, err)
	assert.Equal(t, 1, webReplicas)

	// worker component: role and replicas explicitly set, overriding defaults
	workerRole, err := cfg.String("components.worker.role")
	require.NoError(t, err)
	assert.Equal(t, "worker", workerRole)

	workerReplicas, err := cfg.Int("components.worker.replicas")
	require.NoError(t, err)
	assert.Equal(t, 3, workerReplicas)
}
