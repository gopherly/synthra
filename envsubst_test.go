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

package synthra

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra/resolve"
	"gopherly.dev/synthra/source"
)

func TestWithEnvSubst_VarsResolver(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"envFile": ".env.${NAME}",
			"cluster": "${REGION}-cluster",
		})),
		WithEnvSubst(resolve.Vars(map[string]string{
			"NAME":   "production",
			"REGION": "eu-west-1",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	envFile, err := cfg.String("envfile")
	require.NoError(t, err)
	assert.Equal(t, ".env.production", envFile)

	cluster, err := cfg.String("cluster")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1-cluster", cluster)
}

func TestWithEnvSubst_DefaultFallback(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"port": "${PORT:-3000}",
		})),
		WithEnvSubst(resolve.Vars(map[string]string{})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	port, err := cfg.String("port")
	require.NoError(t, err)
	assert.Equal(t, "3000", port)
}

func TestWithEnvSubst_DefaultFallbackOverriddenByVar(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"port": "${PORT:-3000}",
		})),
		WithEnvSubst(resolve.Vars(map[string]string{
			"PORT": "9090",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	port, err := cfg.String("port")
	require.NoError(t, err)
	assert.Equal(t, "9090", port)
}

func TestWithEnvSubst_OSResolver(t *testing.T) {
	t.Setenv("SYNTHRA_ENVSUBST_HOST", "os.example.com")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"host": "${SYNTHRA_ENVSUBST_HOST}",
		})),
		WithEnvSubst(resolve.OS()),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	host, err := cfg.String("host")
	require.NoError(t, err)
	assert.Equal(t, "os.example.com", host)
}

func TestWithEnvSubst_OSPrefixResolver(t *testing.T) {
	t.Setenv("DPY_VAR_PORT", "7070")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"port": "${PORT}",
		})),
		WithEnvSubst(resolve.OSPrefix("DPY_VAR_")),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	port, err := cfg.String("port")
	require.NoError(t, err)
	assert.Equal(t, "7070", port)
}

func TestWithEnvSubst_LayeredPriority(t *testing.T) {
	t.Setenv("APP_PORT", "9999")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"port": "${PORT:-3000}",
			"host": "${HOST}",
		})),
		WithEnvSubst(
			resolve.Vars(map[string]string{"PORT": "3000", "HOST": "default.local"}),
			resolve.Vars(map[string]string{"PORT": "5000"}),
			resolve.OSPrefix("APP_"),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// APP_PORT=9999 wins (OSPrefix is highest priority)
	port, err := cfg.String("port")
	require.NoError(t, err)
	assert.Equal(t, "9999", port)

	// HOST is only in the first Vars, no APP_HOST or second Vars override
	host, err := cfg.String("host")
	require.NoError(t, err)
	assert.Equal(t, "default.local", host)
}

func TestWithEnvSubst_NestedMap(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"server": map[string]any{
				"host": "${HOST}.example.com",
				"port": "${PORT:-8080}",
			},
		})),
		WithEnvSubst(resolve.Vars(map[string]string{
			"HOST": "api",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	host, err := cfg.String("server.host")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com", host)

	port, err := cfg.String("server.port")
	require.NoError(t, err)
	assert.Equal(t, "8080", port)
}

func TestWithEnvSubst_SliceValues(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"files": []any{".env.${NAME}", "config.${NAME}.yaml"},
		})),
		WithEnvSubst(resolve.Vars(map[string]string{
			"NAME": "staging",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	files, err := cfg.StringSlice("files")
	require.NoError(t, err)
	assert.Equal(t, []string{".env.staging", "config.staging.yaml"}, files)
}

func TestWithEnvSubst_NestedSliceAndMapInSlice(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"items": []any{
				map[string]any{"dsn": "postgres://${HOST}"},
				[]any{"${ENV}-node1", "${ENV}-node2"},
			},
		})),
		WithEnvSubst(resolve.Vars(map[string]string{
			"ENV":  "prod",
			"HOST": "db.prod.local",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	items := cfg.Get("items")
	outerSlice, ok := items.([]any)
	require.True(t, ok)
	require.Len(t, outerSlice, 2)

	mapElem, ok := outerSlice[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "postgres://db.prod.local", mapElem["dsn"])

	innerSlice, ok := outerSlice[1].([]any)
	require.True(t, ok)
	require.Len(t, innerSlice, 2)
	assert.Equal(t, "prod-node1", innerSlice[0])
	assert.Equal(t, "prod-node2", innerSlice[1])
}

func TestWithEnvSubst_UnknownVarWithoutDefaultErrors(t *testing.T) {
	t.Parallel()

	// ${VAR} without a default causes an error when VAR is not resolved.
	// This is standard POSIX strict-mode behavior.
	// Use ${VAR:-} for optional vars that should be empty when unset,
	// or ${VAR:-fallback} for a non-empty fallback.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"path": "/var/${UNKNOWN_VAR_XYZ_STRICT}/config",
		})),
		WithEnvSubst(resolve.Vars(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "UNKNOWN_VAR_XYZ_STRICT")
}

func TestWithEnvSubst_UnknownVarWithEmptyDefault(t *testing.T) {
	t.Parallel()

	// Use ${VAR:-} to make a variable optional (expands to empty string
	// when the variable is not set).
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"path": "/var/${OPTIONAL_VAR:-}/config",
		})),
		WithEnvSubst(resolve.Vars(map[string]string{})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	path, err := cfg.String("path")
	require.NoError(t, err)
	assert.Equal(t, "/var//config", path)
}

func TestWithEnvSubst_NoResolvers(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"port": "8080",
		})),
		WithEnvSubst(),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	port, err := cfg.String("port")
	require.NoError(t, err)
	assert.Equal(t, "8080", port)
}

func TestWithEnvSubst_WorksWithSchemaDefaults(t *testing.T) {
	t.Parallel()

	// Schema default provides the template; envsubst fills it in.
	schema := []byte(`{
		"type": "object",
		"properties": {
			"envfile": {"type": "string", "default": ".env.${NAME}"}
		}
	}`)

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{})),
		WithJSONSchema(schema),
		WithEnvSubst(resolve.Vars(map[string]string{
			"NAME": "production",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	envFile, err := cfg.String("envfile")
	require.NoError(t, err)
	assert.Equal(t, ".env.production", envFile)
}

func TestWithEnvSubst_PosixUppercase(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"region": "${REGION^^}",
		})),
		WithEnvSubst(resolve.Vars(map[string]string{
			"REGION": "eu-west-1",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	region, err := cfg.String("region")
	require.NoError(t, err)
	assert.Equal(t, "EU-WEST-1", region)
}

func TestWithEnvSubst_OSReadsLiveEnv(t *testing.T) {
	t.Setenv("SYNTHRA_LIVE_ENVSUBST", "first")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"val": "${SYNTHRA_LIVE_ENVSUBST}",
		})),
		WithEnvSubst(resolve.OS()),
	)
	require.NoError(t, err)

	require.NoError(t, cfg.Load(context.Background()))
	val, err := cfg.String("val")
	require.NoError(t, err)
	assert.Equal(t, "first", val)

	// Change the env var and reload.
	t.Setenv("SYNTHRA_LIVE_ENVSUBST", "second")
	require.NoError(t, cfg.Load(context.Background()))
	val, err = cfg.String("val")
	require.NoError(t, err)
	assert.Equal(t, "second", val)
}
