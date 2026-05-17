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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra/source"
)

func TestWithTransform_NilRejected(t *testing.T) {
	t.Parallel()

	_, err := New(WithTransform(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transform function cannot be nil")
}

func TestWithTransform_Single(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"log_level": "WARN",
		})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			if level, ok := values["log_level"].(string); ok {
				values["log_level"] = strings.ToLower(level)
			}
			return values, nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	level, err := cfg.String("log_level")
	require.NoError(t, err)
	assert.Equal(t, "warn", level)
}

func TestWithTransform_Chained(t *testing.T) {
	t.Parallel()

	// First transform lowercases; second appends a suffix.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"env": "PROD",
		})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			if v, ok := values["env"].(string); ok {
				values["env"] = strings.ToLower(v)
			}
			return values, nil
		}),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			if v, ok := values["env"].(string); ok {
				values["env"] = v + "-cluster"
			}
			return values, nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	env, err := cfg.String("env")
	require.NoError(t, err)
	assert.Equal(t, "prod-cluster", env)
}

func TestWithTransform_ErrorAbortsLoad(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("transform failed")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			return nil, sentinel
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "transform[0]", ce.Path)
	assert.ErrorIs(t, loadErr, sentinel)
}

func TestWithTransform_SecondTransformErrorHasCorrectIndex(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("second transform failed")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			return values, nil // first is fine
		}),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			return nil, sentinel
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, "transform[1]", ce.Path)
}

func TestWithTransform_NilReturnTreatedAsEmpty(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithTransform(func(_ map[string]any) (map[string]any, error) {
			var out map[string]any
			return out, nil // return nil map with no error
		}),
	)
	require.NoError(t, err)
	// Load should succeed; nil map becomes empty map
	require.NoError(t, cfg.Load(context.Background()))
}

func TestWithTransform_RunsAfterSchemaDefaults(t *testing.T) {
	t.Parallel()

	// Schema provides default "info" for log_level.
	// Transform should see that default and can modify it.
	schema := []byte(`{
		"type": "object",
		"properties": {
			"log_level": {"type": "string", "default": "info"}
		}
	}`)

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{})),
		WithJSONSchema(schema),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			if level, ok := values["log_level"].(string); ok {
				values["log_level"] = strings.ToUpper(level)
			}
			return values, nil
		}),
	)
	require.NoError(t, err)
	// Note: the schema requires log_level to be one of lowercase values only
	// so we use a permissive schema here.
	require.NoError(t, cfg.Load(context.Background()))

	level, err := cfg.String("log_level")
	require.NoError(t, err)
	assert.Equal(t, "INFO", level)
}

// --- WithInterpolation ---

func TestWithInterpolation_Basic(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"envFile": ".env.{name}",
			"cluster": "{region}-cluster",
		})),
		WithInterpolation(map[string]string{
			"name":   "production",
			"region": "eu-west-1",
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	envFile, err := cfg.String("envFile")
	require.NoError(t, err)
	assert.Equal(t, ".env.production", envFile)

	cluster, err := cfg.String("cluster")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1-cluster", cluster)
}

func TestWithInterpolation_NestedValues(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"server": map[string]any{
				"host": "{hostname}.example.com",
			},
		})),
		WithInterpolation(map[string]string{
			"hostname": "api",
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	host, err := cfg.String("server.host")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com", host)
}

func TestWithInterpolation_UnmatchedPlaceholderLeftAsIs(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"path": "/var/{unknown}/config",
		})),
		WithInterpolation(map[string]string{
			"name": "prod",
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	path, err := cfg.String("path")
	require.NoError(t, err)
	assert.Equal(t, "/var/{unknown}/config", path)
}

func TestWithInterpolation_EmptyVarsIsNoOp(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"path": "/var/{name}/config",
		})),
		WithInterpolation(map[string]string{}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	path, err := cfg.String("path")
	require.NoError(t, err)
	assert.Equal(t, "/var/{name}/config", path)
}

func TestWithInterpolation_ArrayValues(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"files": []any{".env.{name}", "config.{name}.yaml"},
		})),
		WithInterpolation(map[string]string{
			"name": "staging",
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	files, err := cfg.StringSlice("files")
	require.NoError(t, err)
	assert.Equal(t, []string{".env.staging", "config.staging.yaml"}, files)
}

func TestWithInterpolation_WorksWithSchemaDefaults(t *testing.T) {
	t.Parallel()

	// Schema default provides the template; interpolation fills it in.
	// Note: synthra normalizes all keys to lowercase, so "envFile" in the
	// schema becomes "envfile" in the loaded map.
	schema := []byte(`{
		"type": "object",
		"properties": {
			"envfile": {"type": "string", "default": ".env.{name}"}
		}
	}`)

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{})),
		WithJSONSchema(schema),
		WithInterpolation(map[string]string{
			"name": "production",
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	envFile, err := cfg.String("envfile")
	require.NoError(t, err)
	assert.Equal(t, ".env.production", envFile)
}

// --- replacePlaceholders unit tests ---

func TestReplacePlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		vars  map[string]string
		want  string
	}{
		{
			name:  "single placeholder replaced",
			input: "hello {name}",
			vars:  map[string]string{"name": "world"},
			want:  "hello world",
		},
		{
			name:  "multiple placeholders replaced",
			input: "{a} and {b}",
			vars:  map[string]string{"a": "foo", "b": "bar"},
			want:  "foo and bar",
		},
		{
			name:  "no placeholder returns unchanged",
			input: "no placeholders here",
			vars:  map[string]string{"name": "world"},
			want:  "no placeholders here",
		},
		{
			name:  "unmatched placeholder left as-is",
			input: "value/{unknown}",
			vars:  map[string]string{"name": "world"},
			want:  "value/{unknown}",
		},
		{
			name:  "empty vars returns unchanged",
			input: "hello {name}",
			vars:  map[string]string{},
			want:  "hello {name}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, replacePlaceholders(tt.input, tt.vars))
		})
	}
}

// TestWithInterpolation_NestedSliceAndMapInSlice covers the two branches in
// interpolateSlice that recurse into []any and map[string]any elements.
func TestWithInterpolation_NestedSliceAndMapInSlice(t *testing.T) {
	t.Parallel()

	vars := map[string]string{
		"env":  "prod",
		"host": "db.prod.local",
	}

	src := source.NewMap(map[string]any{
		"items": []any{
			// Branch 1: map inside slice
			map[string]any{"dsn": "postgres://{host}"},
			// Branch 2: nested slice inside slice
			[]any{"{env}-node1", "{env}-node2"},
		},
	})

	cfg, err := New(WithSource(src), WithInterpolation(vars))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	items := cfg.Get("items")

	outerSlice, ok := items.([]any)
	require.True(t, ok, "expected []any")
	require.Len(t, outerSlice, 2)

	// Branch 1: map element
	mapElem, ok := outerSlice[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "postgres://db.prod.local", mapElem["dsn"])

	// Branch 2: nested slice element
	innerSlice, ok := outerSlice[1].([]any)
	require.True(t, ok)
	require.Len(t, innerSlice, 2)
	assert.Equal(t, "prod-node1", innerSlice[0])
	assert.Equal(t, "prod-node2", innerSlice[1])
}
