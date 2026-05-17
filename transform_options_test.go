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
	require.NoError(t, cfg.Load(context.Background()))

	level, err := cfg.String("log_level")
	require.NoError(t, err)
	assert.Equal(t, "INFO", level)
}
