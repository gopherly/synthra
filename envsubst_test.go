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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra/source"
)

func TestWithEnvSubst_VarsResolver(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"envFile": ".env.${NAME}",
			"cluster": "${REGION}-cluster",
		})),
		WithEnvSubst(FromMap(map[string]string{
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
		WithEnvSubst(FromMap(map[string]string{})),
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
		WithEnvSubst(FromMap(map[string]string{
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
		WithEnvSubst(FromEnv()),
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
		WithEnvSubst(FromEnv().Prefix("DPY_VAR_")),
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
		// First match wins: APP_PORT=9999 takes priority over the middle map's
		// PORT=5000, which itself takes priority over the defaults map.
		WithEnvSubst(
			FromEnv().Prefix("APP_"). // highest: APP_*
							Or(FromMap(map[string]string{"PORT": "5000"})).                          // middle
							Or(FromMap(map[string]string{"PORT": "3000", "HOST": "default.local"})), // lowest
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// APP_PORT=9999 wins (first match, highest priority)
	port, err := cfg.String("port")
	require.NoError(t, err)
	assert.Equal(t, "9999", port)

	// HOST is only in the lowest-priority map; falls through to it
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
		WithEnvSubst(FromMap(map[string]string{
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
		WithEnvSubst(FromMap(map[string]string{
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
		WithEnvSubst(FromMap(map[string]string{
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
		WithEnvSubst(FromMap(map[string]string{})),
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
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	path, err := cfg.String("path")
	require.NoError(t, err)
	assert.Equal(t, "/var//config", path)
}

func TestWithEnvSubst_NilResolverErrors(t *testing.T) {
	t.Parallel()

	_, err := New(
		WithSource(source.NewMap(map[string]any{})),
		WithEnvSubst(nil),
	)
	require.Error(t, err)

	var ce *ConfigError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, OpNew, ce.Op)
	assert.Equal(t, "WithEnvSubst", ce.Path)
}

func TestWithEnvSubst_EmptyStringIsFound(t *testing.T) {
	t.Parallel()

	// A resolver that returns ("", true) for VAR is "found" even though the
	// value is empty. The Or chain must stop there — the fallback resolver must
	// NOT be consulted.
	//
	// Note: ${VAR:-default} is POSIX "if unset OR empty", so the underlying
	// envsubst library will still substitute the default even when the resolver
	// returns ("", true). The short-circuit guarantee of Or is that no further
	// resolver in the chain is called — not that the envsubst :-default syntax
	// is suppressed. To observe that, we use bare ${VAR} (no default) so that
	// the expansion simply produces "".
	explicit := Resolver(func(name string) (string, bool) {
		if name == "VAR" {
			return "", true
		}
		return "", false
	})
	// fallback would return "from-fallback" if it were ever called
	fallback := FromMap(map[string]string{"VAR": "from-fallback"})

	r := explicit.Or(fallback)

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"val": "${VAR}", // bare expansion — no :-default
		})),
		WithEnvSubst(r),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	val, err := cfg.String("val")
	require.NoError(t, err)
	// If Or had fallen through to fallback, val would be "from-fallback".
	// "" confirms Or short-circuited at the explicit resolver.
	assert.Equal(t, "", val)
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
		WithEnvSubst(FromMap(map[string]string{
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
		WithEnvSubst(FromMap(map[string]string{
			"REGION": "eu-west-1",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	region, err := cfg.String("region")
	require.NoError(t, err)
	assert.Equal(t, "EU-WEST-1", region)
}

func TestWithEnvSubst_NonStringValuesPassThrough(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"port":    8080,
			"enabled": true,
			"ratio":   0.75,
			"nothing": nil,
			"name":    "${NAME}",
		})),
		WithEnvSubst(FromMap(map[string]string{
			"NAME": "test",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	assert.Equal(t, 8080, cfg.Get("port"))
	assert.Equal(t, true, cfg.Get("enabled"))
	assert.Equal(t, 0.75, cfg.Get("ratio"))
	assert.Nil(t, cfg.Get("nothing"))

	name, err := cfg.String("name")
	require.NoError(t, err)
	assert.Equal(t, "test", name)
}

func TestWithEnvSubst_NonStringValuesInSlicePassThrough(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"mixed": []any{42, true, 3.14, nil, "${TAG}"},
		})),
		WithEnvSubst(FromMap(map[string]string{
			"TAG": "latest",
		})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	items := cfg.Get("mixed")
	slice, ok := items.([]any)
	require.True(t, ok)
	require.Len(t, slice, 5)
	assert.Equal(t, 42, slice[0])
	assert.Equal(t, true, slice[1])
	assert.Equal(t, 3.14, slice[2])
	assert.Nil(t, slice[3])
	assert.Equal(t, "latest", slice[4])
}

func TestWithEnvSubst_ErrorInSliceStringElement(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"hosts": []any{"ok.example.com", "${MISSING_SLICE_VAR}"},
		})),
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "MISSING_SLICE_VAR")
	assert.Contains(t, loadErr.Error(), `key "hosts[1]"`)
}

func TestWithEnvSubst_ErrorFromNestedMapInSlice(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"items": []any{
				map[string]any{"dsn": "${MISSING_DSN_VAR}"},
			},
		})),
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "MISSING_DSN_VAR")
	assert.Contains(t, loadErr.Error(), `key "items[0].dsn"`)
}

func TestWithEnvSubst_ErrorFromNestedSliceInSlice(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"matrix": []any{
				[]any{"${MISSING_MATRIX_VAR}"},
			},
		})),
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "MISSING_MATRIX_VAR")
	assert.Contains(t, loadErr.Error(), `key "matrix[0][0]"`)
}

func TestWithEnvSubst_ErrorFromNestedMapInMap(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"db": map[string]any{
				"primary": map[string]any{
					"host": "${MISSING_DB_HOST}",
				},
			},
		})),
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "MISSING_DB_HOST")
	assert.Contains(t, loadErr.Error(), `key "db.primary.host"`)
}

func TestWithEnvSubst_ErrorFromSliceInMap(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"tags": []any{"${MISSING_TAG_VAR}"},
		})),
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "MISSING_TAG_VAR")
	assert.Contains(t, loadErr.Error(), `key "tags[0]"`)
}

func TestWithEnvSubst_ErrorWrappedAsConfigError(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"path": "${MISSING_CFG_ERR_VAR}",
		})),
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "step[0]:transform", ce.Path)
	assert.Contains(t, ce.Err.Error(), "envsubst:")
	assert.Contains(t, ce.Err.Error(), `key "path"`)
	assert.Contains(t, ce.Err.Error(), "MISSING_CFG_ERR_VAR")
}

func TestWithEnvSubst_ErrorIndexWhenCombinedWithTransform(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"host": "${MISSING_COMBINED_VAR}",
		})),
		WithTransform(func(_ *Values) error {
			return nil
		}),
		WithEnvSubst(FromMap(map[string]string{})),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "step[1]:transform", ce.Path)
}

func TestWithEnvSubst_EmptyMapInput(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{})),
		WithEnvSubst(FromMap(map[string]string{"KEY": "val"})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
}

func TestWithEnvSubst_EmptySliceInMap(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"items": []any{},
		})),
		WithEnvSubst(FromMap(map[string]string{"KEY": "val"})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	items := cfg.Get("items")
	slice, ok := items.([]any)
	require.True(t, ok)
	assert.Empty(t, slice)
}

func TestWithEnvSubst_OSReadsLiveEnv(t *testing.T) {
	t.Setenv("SYNTHRA_LIVE_ENVSUBST", "first")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"val": "${SYNTHRA_LIVE_ENVSUBST}",
		})),
		WithEnvSubst(FromEnv()),
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

// --- WithEnvSubstFunc tests ---

func TestWithEnvSubstFunc_Success(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"greeting": "hello ${NAME}",
		})),
		WithEnvSubstFunc(func(_ *Values) (Resolver, error) {
			return FromMap(map[string]string{"NAME": "world"}), nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	greeting, err := cfg.String("greeting")
	require.NoError(t, err)
	assert.Equal(t, "hello world", greeting)
}

func TestWithEnvSubstFunc_CallbackError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("resolver setup failed")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"val": "${X}",
		})),
		WithEnvSubstFunc(func(_ *Values) (Resolver, error) {
			return nil, sentinel
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.ErrorIs(t, loadErr, sentinel)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "step[0]:transform", ce.Path)
}

func TestWithEnvSubstFunc_DynamicEnvFile(t *testing.T) {
	// Write a .env file to a temp dir.
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.production")
	require.NoError(t, os.WriteFile(envPath, []byte("REGION=eu-west-1\n"), 0o600))

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"envfile": envPath,
			"cluster": "${REGION}-cluster",
		})),
		WithEnvSubstFunc(func(v *Values) (Resolver, error) {
			path := v.StringOr("envfile", "")
			if path == "" {
				return FromEnv(), nil
			}
			envFile, err := FromEnvFile(path)
			if err != nil {
				return nil, err
			}
			// Compose: OS env takes priority, .env file is the fallback.
			return FromEnv().Or(envFile), nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	cluster, err := cfg.String("cluster")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1-cluster", cluster)
}

func TestWithEnvSubstFunc_DynamicEnvFile_OSEnvTakesPriority(t *testing.T) {
	// OS env overrides the .env file value when composed with Or.
	t.Setenv("REGION", "us-east-1")

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("REGION=eu-west-1\n"), 0o600))

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"envfile": envPath,
			"cluster": "${REGION}-cluster",
		})),
		WithEnvSubstFunc(func(v *Values) (Resolver, error) {
			envFile, err := FromEnvFile(v.StringOr("envfile", ""))
			if err != nil {
				return nil, err
			}
			return FromEnv().Or(envFile), nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	cluster, err := cfg.String("cluster")
	require.NoError(t, err)
	// OS env (us-east-1) wins over .env file (eu-west-1)
	assert.Equal(t, "us-east-1-cluster", cluster)
}

func TestWithEnvSubstFunc_ReceivesCurrentValues(t *testing.T) {
	t.Parallel()

	var gotValues map[string]any

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"env":  "production",
			"host": "${HOST}",
		})),
		WithEnvSubstFunc(func(v *Values) (Resolver, error) {
			gotValues = v.Raw()
			return FromMap(map[string]string{"HOST": "api.example.com"}), nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// The callback received the merged values map.
	require.NotNil(t, gotValues)
	assert.Equal(t, "production", gotValues["env"])

	host, err := cfg.String("host")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com", host)
}

func TestWithEnvSubstFunc_NilFuncErrors(t *testing.T) {
	t.Parallel()

	_, err := New(
		WithSource(source.NewMap(map[string]any{})),
		WithEnvSubstFunc(nil),
	)
	require.Error(t, err)

	var ce *ConfigError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, OpNew, ce.Op)
	assert.Equal(t, "WithEnvSubstFunc", ce.Path)
}

func TestWithEnvSubstFunc_ErrorWrappedAsConfigError(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"val": "${X}",
		})),
		WithEnvSubstFunc(func(_ *Values) (Resolver, error) {
			return nil, fmt.Errorf("setup: %w", errors.New("connection refused"))
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Contains(t, ce.Err.Error(), "envsubst:")
	assert.Contains(t, ce.Err.Error(), "connection refused")
}
