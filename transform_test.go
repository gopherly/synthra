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
		WithTransform(func(_ context.Context, v *Configurable) error {
			if level, err := v.String("log_level"); err == nil {
				return v.Set("log_level", strings.ToLower(level))
			}
			return nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	level, err := cfg.String("log_level")
	require.NoError(t, err)
	assert.Equal(t, "warn", level)
}

func TestWithTransform_GetSetWithFoldMatch(t *testing.T) {
	t.Parallel()

	// Source uses mixed-case key; transform reads via fold-insensitive Get.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"LogLevel": "DEBUG",
		})),
		WithTransform(func(_ context.Context, v *Configurable) error {
			level := v.StringOr("loglevel", "")
			if level != "" {
				return v.Set("loglevel", strings.ToLower(level))
			}
			return nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// Both casings should return the lowercased value.
	level, err := cfg.String("LogLevel")
	require.NoError(t, err)
	assert.Equal(t, "debug", level)
}

func TestWithTransform_WalkReplacesValues(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"env": "PROD",
		})),
		WithTransform(func(_ context.Context, v *Configurable) error {
			v.Walk(func(_ string, val any) (any, bool) {
				if s, ok := val.(string); ok {
					return strings.ToLower(s), true
				}
				return nil, false
			})
			return nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	env, err := cfg.String("env")
	require.NoError(t, err)
	assert.Equal(t, "prod", env)
}

func TestWithTransform_Chained(t *testing.T) {
	t.Parallel()

	// First transform lowercases; second appends a suffix.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"env": "PROD",
		})),
		WithTransform(func(_ context.Context, v *Configurable) error {
			if s, err := v.String("env"); err == nil {
				return v.Set("env", strings.ToLower(s))
			}
			return nil
		}),
		WithTransform(func(_ context.Context, v *Configurable) error {
			if s, err := v.String("env"); err == nil {
				return v.Set("env", s+"-cluster")
			}
			return nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	env, err := cfg.String("env")
	require.NoError(t, err)
	assert.Equal(t, "prod-cluster", env)
}

func TestWithTransform_NilFunctionRejected(t *testing.T) {
	t.Parallel()

	_, err := New(WithTransform(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestWithTransform_ErrorAbortsLoad(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("transform failed")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithTransform(func(_ context.Context, _ *Configurable) error {
			return sentinel
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
		WithTransform(func(_ context.Context, _ *Configurable) error {
			return nil
		}),
		WithTransform(func(_ context.Context, _ *Configurable) error {
			return sentinel
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
		WithTransform(func(_ context.Context, v *Configurable) error {
			// Delete everything to simulate an effectively empty result.
			_ = v.Delete("key")
			return nil
		}),
	)
	require.NoError(t, err)
	// Load should succeed; deleted key is simply gone.
	require.NoError(t, cfg.Load(context.Background()))
	assert.Nil(t, cfg.Get("key"))
}

func TestWithTransform_AddsNewKey(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"host": "example.com",
			"port": "443",
		})),
		WithTransform(func(_ context.Context, v *Configurable) error {
			host := v.StringOr("host", "")
			port := v.StringOr("port", "")
			if host == "" || port == "" {
				return errors.New("host or port missing")
			}
			return v.Set("address", host+":"+port)
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
		WithTransform(func(_ context.Context, v *Configurable) error {
			v.Delete("remove")
			return nil
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
		WithTransform(func(_ context.Context, v *Configurable) error {
			return v.Set("derived", "from-first")
		}),
		WithTransform(func(_ context.Context, v *Configurable) error {
			existing := v.StringOr("derived", "")
			if existing != "" {
				return v.Set("derived", existing+"-modified")
			}
			return nil
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
		WithTransform(func(_ context.Context, v *Configurable) error {
			return v.Set("injected", "hello")
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
		WithEnvSubstFunc(func(_ context.Context, _ *Configurable) (Resolver, error) {
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
		WithTransform(func(_ context.Context, v *Configurable) error {
			if level, err := v.String("log_level"); err == nil {
				return v.Set("log_level", strings.ToUpper(level))
			}
			return nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	level, err := cfg.String("log_level")
	require.NoError(t, err)
	assert.Equal(t, "INFO", level)
}

func TestWithEnvSubst_SingleResolver(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"greeting": "hello ${NAME}",
		})),
		WithEnvSubst(FromMap(map[string]string{"NAME": "world"})),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	greeting, err := cfg.String("greeting")
	require.NoError(t, err)
	assert.Equal(t, "hello world", greeting)
}

func TestWithEnvSubstFunc_ReceivesTypedValues(t *testing.T) {
	t.Parallel()

	var gotEnv string

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"env":     "production",
			"cluster": "${REGION}-cluster",
		})),
		WithEnvSubstFunc(func(_ context.Context, v *Configurable) (Resolver, error) {
			gotEnv = v.StringOr("env", "")
			return FromMap(map[string]string{"REGION": "eu-west-1"}), nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	assert.Equal(t, "production", gotEnv)

	cluster, err := cfg.String("cluster")
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1-cluster", cluster)
}
