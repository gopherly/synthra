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

// schemaForVersion returns a minimal versioned schema as raw bytes.
func schemaForVersion(version string) []byte {
	switch version {
	case "v1":
		return []byte(`{
			"type": "object",
			"required": ["apiVersion"],
			"properties": {
				"apiVersion": {"type": "string"},
				"port":       {"type": "integer", "default": 8080}
			}
		}`)
	case "v2":
		return []byte(`{
			"type": "object",
			"required": ["apiVersion"],
			"properties": {
				"apiVersion": {"type": "string"},
				"port":       {"type": "integer", "default": 9090},
				"log_level":  {"type": "string",  "default": "debug"}
			}
		}`)
	default:
		return nil
	}
}

// versionSelector is a reusable selector that maps the "apiVersion" key to
// the correct schema bytes using schemaForVersion above.
func versionSelector(_ context.Context, v *Configurable) ([]byte, error) {
	ver := v.StringOr("apiVersion", "")
	if ver == "" {
		return nil, errors.New("apiVersion is required")
	}
	schema := schemaForVersion(ver)
	if schema == nil {
		return nil, errors.New("unknown apiVersion: " + ver)
	}
	return schema, nil
}

func TestWithJSONSchemaFunc_NilRejected(t *testing.T) {
	t.Parallel()

	_, err := New(WithJSONSchemaFunc(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selector cannot be nil")
}

func TestWithJSONSchemaFunc_HappyPath(t *testing.T) {
	t.Parallel()

	var selectorGotVersion string

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiVersion": "v1",
		})),
		WithJSONSchemaFunc(func(ctx context.Context, v *Configurable) ([]byte, error) {
			selectorGotVersion = v.StringOr("apiVersion", "")
			return versionSelector(ctx, v)
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// Selector received the merged values.
	assert.Equal(t, "v1", selectorGotVersion)

	// Schema default for v1 (port=8080) applied.
	port, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, port)
}

func TestWithJSONSchemaFunc_SelectsCorrectSchemaByVersion(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		version      string
		wantPort     int
		wantLogLevel string
	}{
		{version: "v1", wantPort: 8080, wantLogLevel: ""},
		{version: "v2", wantPort: 9090, wantLogLevel: "debug"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()

			cfg, err := New(
				WithSource(source.NewMap(map[string]any{
					"apiVersion": tc.version,
				})),
				WithJSONSchemaFunc(versionSelector),
			)
			require.NoError(t, err)
			require.NoError(t, cfg.Load(context.Background()))

			port, err := cfg.Int("port")
			require.NoError(t, err)
			assert.Equal(t, tc.wantPort, port)

			if tc.wantLogLevel != "" {
				level, levelErr := cfg.String("log_level")
				require.NoError(t, levelErr)
				assert.Equal(t, tc.wantLogLevel, level)
			}
		})
	}
}

func TestWithJSONSchemaFunc_SelectorErrorAbortsLoad(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("selector failed")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) {
			return nil, sentinel
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "step[0]:schema", ce.Path)
	assert.ErrorIs(t, loadErr, sentinel)
}

func TestWithJSONSchemaFunc_InvalidSchemaBytesAbortsLoad(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) {
			return []byte(`not valid json`), nil
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "step[0]:schema", ce.Path)
}

func TestWithJSONSchemaFunc_UnknownVersionReturnsDescriptiveError(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiVersion": "v99",
		})),
		WithJSONSchemaFunc(versionSelector),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "unknown apiVersion")
}

func TestWithJSONSchemaFunc_DefaultsFromSelectedSchema(t *testing.T) {
	t.Parallel()

	// v2 schema has defaults for both port and log_level.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiVersion": "v2",
		})),
		WithJSONSchemaFunc(versionSelector),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	port, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 9090, port)

	level, err := cfg.String("log_level")
	require.NoError(t, err)
	assert.Equal(t, "debug", level)
}

func TestWithJSONSchemaFunc_UserValueOverridesSchemaDefault(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiVersion": "v1",
			"port":       7777,
		})),
		WithJSONSchemaFunc(versionSelector),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	port, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 7777, port)
}

func TestWithJSONSchemaFunc_TransformRunsAfterSchema(t *testing.T) {
	t.Parallel()

	// The transform uppercases the log_level which comes from a v2 schema default.
	// WithJSONSchemaFunc runs first (applying defaults + validating), then the
	// transform uppercases the result.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiVersion": "v2",
		})),
		WithJSONSchemaFunc(versionSelector),
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
	// v2 default is "debug"; transform uppercases it to "DEBUG"
	assert.Equal(t, "DEBUG", level)
}

func TestWithJSONSchemaFunc_ValidationFailsForInvalidValue(t *testing.T) {
	t.Parallel()

	// Schema requires "port" to be an integer; supply a string to trigger
	// a validation failure.
	invalidSchema := []byte(`{
		"type": "object",
		"properties": {
			"apiVersion": {"type": "string"},
			"port": {"type": "integer"}
		},
		"required": ["port"]
	}`)

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiVersion": "v1",
			"port":       "not-an-integer",
		})),
		WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) {
			return invalidSchema, nil
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "step[0]:schema", ce.Path)
}

func TestWithJSONSchemaFunc_ConcurrentLoadIsSafe(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiVersion": "v1",
		})),
		WithJSONSchemaFunc(versionSelector),
	)
	require.NoError(t, err)

	const goroutines = 10
	errs := make(chan error, goroutines)

	for range goroutines {
		go func() {
			errs <- cfg.Load(context.Background())
		}()
	}

	for range goroutines {
		assert.NoError(t, <-errs)
	}
}

// TestWithJSONSchemaFunc_MultipleSchemas verifies that multiple WithJSONSchemaFunc
// calls each add an independent schema step, both running in registration order.
func TestWithJSONSchemaFunc_MultipleSchemas(t *testing.T) {
	t.Parallel()

	partialSchema := []byte(`{
		"type": "object",
		"required": ["apiVersion"]
	}`)
	fullSchema := []byte(`{
		"type": "object",
		"required": ["apiVersion", "port"],
		"properties": {
			"port": {"type": "integer"}
		}
	}`)

	t.Run("both schemas pass", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{
				"apiVersion": "v1",
				"port":       8080,
			})),
			WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) { return partialSchema, nil }),
			WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) { return fullSchema, nil }),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))
	})

	t.Run("first schema fails at step[0]", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{
				// missing apiVersion — fails first schema
				"port": 8080,
			})),
			WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) { return partialSchema, nil }),
			WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) { return fullSchema, nil }),
		)
		require.NoError(t, err)
		loadErr := cfg.Load(context.Background())
		require.Error(t, loadErr)
		var ce *ConfigError
		require.ErrorAs(t, loadErr, &ce)
		assert.Equal(t, "step[0]:schema", ce.Path)
	})

	t.Run("second schema fails at step[1]", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{
				"apiVersion": "v1",
				// missing port — passes partialSchema, fails fullSchema
			})),
			WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) { return partialSchema, nil }),
			WithJSONSchemaFunc(func(_ context.Context, _ *Configurable) ([]byte, error) { return fullSchema, nil }),
		)
		require.NoError(t, err)
		loadErr := cfg.Load(context.Background())
		require.Error(t, loadErr)
		var ce *ConfigError
		require.ErrorAs(t, loadErr, &ce)
		assert.Equal(t, "step[1]:schema", ce.Path)
	})
}
