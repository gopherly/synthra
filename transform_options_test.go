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
	assert.Equal(t, "step[0]:transform", ce.Path)
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
	assert.Equal(t, "step[1]:transform", ce.Path)
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

func TestWithTransform_AddsNewKey(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"host": "example.com",
			"port": "443",
		})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			host, ok := values["host"].(string)
			if !ok {
				return nil, errors.New("host is not a string")
			}
			port, ok := values["port"].(string)
			if !ok {
				return nil, errors.New("port is not a string")
			}
			values["address"] = host + ":" + port
			return values, nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	addr, err := cfg.String("address")
	require.NoError(t, err)
	assert.Equal(t, "example.com:443", addr)
}

func TestWithTransform_RemovesKey(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"keep":   "yes",
			"remove": "gone",
		})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			delete(values, "remove")
			return values, nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	keep, err := cfg.String("keep")
	require.NoError(t, err)
	assert.Equal(t, "yes", keep)
	assert.Nil(t, cfg.Get("remove"))
}

func TestWithTransform_PipelineSeesAddedKey(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"base": "value",
		})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			values["derived"] = "from-first"
			return values, nil
		}),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			if v, ok := values["derived"].(string); ok {
				values["derived"] = v + "-modified"
			}
			return values, nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	derived, err := cfg.String("derived")
	require.NoError(t, err)
	assert.Equal(t, "from-first-modified", derived)
}

func TestWithTransform_EmptySourceMap(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{})),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			values["injected"] = "hello"
			return values, nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	val, err := cfg.String("injected")
	require.NoError(t, err)
	assert.Equal(t, "hello", val)
}

func TestWithEnvSubstFunc_SubstitutionError(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"key": "${UNCLOSED",
		})),
		WithEnvSubstFunc(func(_ map[string]any) (Resolver, error) {
			return FromMap(map[string]string{}), nil
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Contains(t, ce.Path, "transform")
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
