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
			"required": ["apiversion"],
			"properties": {
				"apiversion": {"type": "string"},
				"port":       {"type": "integer", "default": 8080}
			}
		}`)
	case "v2":
		return []byte(`{
			"type": "object",
			"required": ["apiversion"],
			"properties": {
				"apiversion": {"type": "string"},
				"port":       {"type": "integer", "default": 9090},
				"log_level":  {"type": "string",  "default": "debug"}
			}
		}`)
	default:
		return nil
	}
}

// versionSelector is a reusable selector that maps the "apiversion" key to
// the correct schema bytes using schemaForVersion above.
func versionSelector(values map[string]any) ([]byte, error) {
	ver, ok := values["apiversion"].(string)
	if !ok || ver == "" {
		return nil, errors.New("apiversion is required")
	}
	schema := schemaForVersion(ver)
	if schema == nil {
		return nil, errors.New("unknown apiversion: " + ver)
	}
	return schema, nil
}

func TestWithJSONSchemaSelector_NilRejected(t *testing.T) {
	t.Parallel()

	_, err := New(WithJSONSchemaSelector(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selector function cannot be nil")
}

func TestWithJSONSchemaSelector_MutualExclusionWithJSONSchema(t *testing.T) {
	t.Parallel()

	_, err := New(
		WithJSONSchema(schemaForVersion("v1")),
		WithJSONSchemaSelector(versionSelector),
	)
	require.Error(t, err)

	var ce *ConfigError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, OpNew, ce.Op)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestWithJSONSchemaSelector_HappyPath(t *testing.T) {
	t.Parallel()

	var selectorGotValues map[string]any

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiversion": "v1",
		})),
		WithJSONSchemaSelector(func(values map[string]any) ([]byte, error) {
			selectorGotValues = values
			return versionSelector(values)
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// Selector received the merged values
	assert.Equal(t, "v1", selectorGotValues["apiversion"])

	// Schema default for v1 (port=8080) applied
	port, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, port)
}

func TestWithJSONSchemaSelector_SelectsCorrectSchemaByVersion(t *testing.T) {
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
					"apiversion": tc.version,
				})),
				WithJSONSchemaSelector(versionSelector),
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

func TestWithJSONSchemaSelector_SelectorErrorAbortsLoad(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("selector failed")

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithJSONSchemaSelector(func(_ map[string]any) ([]byte, error) {
			return nil, sentinel
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "json-schema-selector", ce.Path)
	assert.ErrorIs(t, loadErr, sentinel)
}

func TestWithJSONSchemaSelector_InvalidSchemaBytesAbortsLoad(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"key": "value"})),
		WithJSONSchemaSelector(func(_ map[string]any) ([]byte, error) {
			return []byte(`not valid json`), nil
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "json-schema-selector", ce.Path)
}

func TestWithJSONSchemaSelector_UnknownVersionReturnsDescriptiveError(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiversion": "v99",
		})),
		WithJSONSchemaSelector(versionSelector),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)
	assert.Contains(t, loadErr.Error(), "unknown apiversion")
}

func TestWithJSONSchemaSelector_DefaultsFromSelectedSchema(t *testing.T) {
	t.Parallel()

	// v2 schema has defaults for both port and log_level.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiversion": "v2",
		})),
		WithJSONSchemaSelector(versionSelector),
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

func TestWithJSONSchemaSelector_UserValueOverridesSchemaDefault(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiversion": "v1",
			"port":       7777,
		})),
		WithJSONSchemaSelector(versionSelector),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	port, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 7777, port)
}

func TestWithJSONSchemaSelector_TransformRunsAfterDefaults(t *testing.T) {
	t.Parallel()

	// The transform uppercases the log_level which comes from a v2 schema default.
	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiversion": "v2",
		})),
		WithJSONSchemaSelector(versionSelector),
		WithTransform(func(values map[string]any) (map[string]any, error) {
			if v, ok := values["log_level"].(string); ok {
				values["log_level"] = strings.ToUpper(v)
			}
			return values, nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	level, err := cfg.String("log_level")
	require.NoError(t, err)
	// v2 default is "debug"; transform uppercases it to "DEBUG"
	assert.Equal(t, "DEBUG", level)
}

func TestWithJSONSchemaSelector_ValidationFailsForInvalidValue(t *testing.T) {
	t.Parallel()

	// v1 schema requires "port" to be an integer; supply a string to trigger
	// a validation failure.
	invalidSchema := []byte(`{
		"type": "object",
		"properties": {
			"apiversion": {"type": "string"},
			"port": {"type": "integer"}
		},
		"required": ["port"]
	}`)

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiversion": "v1",
			"port":       "not-an-integer",
		})),
		WithJSONSchemaSelector(func(_ map[string]any) ([]byte, error) {
			return invalidSchema, nil
		}),
	)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, OpLoad, ce.Op)
	assert.Equal(t, "json-schema", ce.Path)
}

func TestWithJSONSchemaSelector_ConcurrentLoadIsSafe(t *testing.T) {
	t.Parallel()

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"apiversion": "v1",
		})),
		WithJSONSchemaSelector(versionSelector),
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
