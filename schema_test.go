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

	"gopherly.dev/synthra/source"
)

func TestApplySchemaDefaults_FixedProperties(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"port": map[string]any{
				"type":    "integer",
				"default": float64(8080),
			},
			"host": map[string]any{
				"type":    "string",
				"default": "localhost",
			},
		},
	}

	t.Run("fills all missing keys", func(t *testing.T) {
		t.Parallel()
		got := applySchemaDefaults(map[string]any{}, schema)
		assert.Equal(t, float64(8080), got["port"])
		assert.Equal(t, "localhost", got["host"])
	})

	t.Run("does not override present keys", func(t *testing.T) {
		t.Parallel()
		got := applySchemaDefaults(map[string]any{
			"port": float64(9090),
		}, schema)
		assert.Equal(t, float64(9090), got["port"])
		assert.Equal(t, "localhost", got["host"])
	})

	t.Run("nil values treated as empty", func(t *testing.T) {
		t.Parallel()
		got := applySchemaDefaults(nil, schema)
		assert.Equal(t, float64(8080), got["port"])
		assert.Equal(t, "localhost", got["host"])
	})
}

func TestApplySchemaDefaults_NestedObjects(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"server": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"port": map[string]any{
						"type":    "integer",
						"default": float64(8080),
					},
					"host": map[string]any{
						"type":    "string",
						"default": "localhost",
					},
				},
			},
		},
	}

	t.Run("fills defaults in nested object", func(t *testing.T) {
		t.Parallel()
		got := applySchemaDefaults(map[string]any{
			"server": map[string]any{},
		}, schema)
		server, ok := got["server"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(8080), server["port"])
		assert.Equal(t, "localhost", server["host"])
	})

	t.Run("does not override nested present keys", func(t *testing.T) {
		t.Parallel()
		got := applySchemaDefaults(map[string]any{
			"server": map[string]any{
				"port": float64(3000),
			},
		}, schema)
		server, ok := got["server"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(3000), server["port"])
		assert.Equal(t, "localhost", server["host"])
	})

	t.Run("creates nested object when absent and has defaults", func(t *testing.T) {
		t.Parallel()
		got := applySchemaDefaults(map[string]any{}, schema)
		server, ok := got["server"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(8080), server["port"])
		assert.Equal(t, "localhost", server["host"])
	})
}

func TestApplySchemaDefaults_PatternProperties(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"components": map[string]any{
				"type": "object",
				"patternProperties": map[string]any{
					"^[a-z0-9-]+$": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"role": map[string]any{
								"type":    "string",
								"default": "service",
							},
							"replicas": map[string]any{
								"type":    "integer",
								"default": float64(1),
							},
						},
					},
				},
			},
		},
	}

	t.Run("applies pattern defaults to all matching keys", func(t *testing.T) {
		t.Parallel()
		values := map[string]any{
			"components": map[string]any{
				"web":    map[string]any{"image": "nginx"},
				"worker": map[string]any{"image": "sidekiq"},
			},
		}
		got := applySchemaDefaults(values, schema)
		components, ok := got["components"].(map[string]any)
		require.True(t, ok)

		web, ok := components["web"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "service", web["role"])
		assert.Equal(t, float64(1), web["replicas"])
		assert.Equal(t, "nginx", web["image"])

		worker, ok := components["worker"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "service", worker["role"])
		assert.Equal(t, float64(1), worker["replicas"])
	})

	t.Run("does not override user-provided values", func(t *testing.T) {
		t.Parallel()
		values := map[string]any{
			"components": map[string]any{
				"worker": map[string]any{
					"image":    "sidekiq",
					"replicas": float64(3),
				},
			},
		}
		got := applySchemaDefaults(values, schema)
		components, ok := got["components"].(map[string]any)
		require.True(t, ok)
		worker, ok := components["worker"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(3), worker["replicas"])
		assert.Equal(t, "service", worker["role"])
	})

	t.Run("ignores keys that do not match pattern", func(t *testing.T) {
		t.Parallel()
		values := map[string]any{
			"components": map[string]any{
				"UPPERCASE": map[string]any{},
			},
		}
		got := applySchemaDefaults(values, schema)
		components, ok := got["components"].(map[string]any)
		require.True(t, ok)
		upper, ok := components["UPPERCASE"].(map[string]any)
		require.True(t, ok)
		// pattern is lowercase-only; UPPERCASE should not have defaults applied
		assert.NotContains(t, upper, "role")
	})
}

func TestApplySchemaDefaults_ArrayItems(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"environments": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"envFile": map[string]any{
							"type":    "string",
							"default": ".env",
						},
					},
				},
			},
		},
	}

	t.Run("fills defaults in array items", func(t *testing.T) {
		t.Parallel()
		values := map[string]any{
			"environments": []any{
				map[string]any{"name": "production"},
				// Use schema-matching casing "envFile".
				map[string]any{"name": "staging", "envFile": ".env.staging"},
			},
		}
		got := applySchemaDefaults(values, schema)
		envs, ok := got["environments"].([]any)
		require.True(t, ok)
		require.Len(t, envs, 2)

		// Schema default key "envFile" is stored with the schema's casing.
		prod, ok := envs[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, ".env", prod["envFile"])

		staging, ok := envs[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, ".env.staging", staging["envFile"])
	})
}

func TestApplySchemaDefaults_NoDefault(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"required_field": map[string]any{
				"type": "string",
				// no "default" key
			},
		},
	}

	t.Run("does not create key when schema has no default", func(t *testing.T) {
		t.Parallel()
		got := applySchemaDefaults(map[string]any{}, schema)
		assert.NotContains(t, got, "required_field")
	})
}

// TestWithJSONSchema_AppliesDefaults verifies the full integration through Load.
func TestWithJSONSchema_AppliesDefaults(t *testing.T) {
	t.Parallel()

	schema := []byte(`{
		"type": "object",
		"properties": {
			"port":      {"type": "integer", "default": 8080},
			"log_level": {"type": "string", "default": "info"}
		}
	}`)

	t.Run("applies schema defaults for missing keys", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{})),
			WithJSONSchema(schema),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))

		port, err := cfg.Int("port")
		require.NoError(t, err)
		assert.Equal(t, 8080, port)

		level, err := cfg.String("log_level")
		require.NoError(t, err)
		assert.Equal(t, "info", level)
	})

	t.Run("source values override schema defaults", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{
				"port": 9090,
			})),
			WithJSONSchema(schema),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))

		port, err := cfg.Int("port")
		require.NoError(t, err)
		assert.Equal(t, 9090, port)
	})
}

// TestWithJSONSchema_PatternPropertyDefaults verifies patternProperties
// defaults are applied to all existing matching keys via Load.
func TestWithJSONSchema_PatternPropertyDefaults(t *testing.T) {
	t.Parallel()

	schema := []byte(`{
		"type": "object",
		"properties": {
			"components": {
				"type": "object",
				"patternProperties": {
					"^[a-z0-9-]+$": {
						"type": "object",
						"properties": {
							"role":     {"type": "string", "default": "service"},
							"replicas": {"type": "integer", "default": 1}
						}
					}
				}
			}
		}
	}`)

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{
			"components": map[string]any{
				"web":    map[string]any{"image": "nginx"},
				"worker": map[string]any{"image": "sidekiq", "replicas": 3},
			},
		})),
		WithJSONSchema(schema),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	webRole, err := cfg.String("components.web.role")
	require.NoError(t, err)
	assert.Equal(t, "service", webRole)

	webReplicas, err := cfg.Int("components.web.replicas")
	require.NoError(t, err)
	assert.Equal(t, 1, webReplicas)

	workerReplicas, err := cfg.Int("components.worker.replicas")
	require.NoError(t, err)
	assert.Equal(t, 3, workerReplicas)

	workerRole, err := cfg.String("components.worker.role")
	require.NoError(t, err)
	assert.Equal(t, "service", workerRole)
}

// TestApplySchemaDefaults_MalformedPropertySchema verifies that the
// propSchemaOk defensive guard silently skips property schema entries that are
// not map[string]any.  This can only happen when the schema map is built
// programmatically — the JSON Schema compiler catches this at build time —
// so we call applySchemaDefaults directly.
func TestApplySchemaDefaults_MalformedPropertySchema(t *testing.T) {
	t.Parallel()

	// Build a schema map where "badKey" has a string value instead of a map.
	schema := map[string]any{
		"properties": map[string]any{
			"validKey": map[string]any{"default": "hello"},
			"badKey":   "this is a string, not a schema map",
		},
	}

	data := map[string]any{}
	result := applySchemaDefaults(data, schema)

	// badKey is silently skipped; validKey still receives its default.
	// The key is stored with the schema's declared casing (not lowercased).
	assert.Equal(t, "hello", result["validKey"])
	assert.NotContains(t, result, "badKey")
}

// TestApplySchemaDefaults_InvalidRegexSkipped verifies the defensive
// [regexp.Compile] error path in applySchemaDefaults.  The JSON Schema compiler
// validates regexes at compile time so this branch is defensive-only; test it
// via the internal function.
func TestApplySchemaDefaults_InvalidRegexSkipped(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"patternProperties": map[string]any{
			// Invalid regex — must be skipped gracefully.
			"[invalid": map[string]any{"default": "should-not-appear"},
			// Valid regex — must still apply.
			"^valid": map[string]any{
				"properties": map[string]any{
					"count": map[string]any{"default": 1},
				},
			},
		},
	}

	data := map[string]any{
		"validGroup": map[string]any{},
	}
	result := applySchemaDefaults(data, schema)

	// Invalid pattern is silently skipped; valid pattern still applies its
	// nested default.
	group, ok := result["validGroup"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, group["count"])
}

// TestApplySchemaDefaults_PatternMatchesNonMap documents the behavior when a
// patternProperties pattern matches a value that is not a map: the scalar is
// discarded and replaced by a new map with defaults applied.  This is the
// defined behavior of applySchemaDefaults.
func TestApplySchemaDefaults_PatternMatchesNonMap(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"patternProperties": map[string]any{
			"^scalar": map[string]any{
				"properties": map[string]any{
					"fallback": map[string]any{"default": "yes"},
				},
			},
		},
	}

	data := map[string]any{
		// Key matches ^scalar but the value is a plain string.
		"scalarKey": "just-a-string",
	}
	result := applySchemaDefaults(data, schema)

	// The string value is replaced by a map with defaults applied.
	resultMap, ok := result["scalarKey"].(map[string]any)
	require.True(t, ok, "expected scalar to be replaced by a map with defaults")
	assert.Equal(t, "yes", resultMap["fallback"])
}

// TestApplySchemaDefaults_InvalidRegexPattern covers the defensive continue that
// skips regex patterns that fail to compile. The JSON Schema compiler rejects
// invalid patterns at construction time, so this path can only be reached when
// calling applySchemaDefaults directly.
func TestApplySchemaDefaults_InvalidRegexPattern(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"patternProperties": map[string]any{
			"[invalid(regex": map[string]any{
				"properties": map[string]any{
					"name": map[string]any{"default": "fallback"},
				},
			},
		},
	}

	values := map[string]any{"somekey": map[string]any{}}
	got := applySchemaDefaults(values, schema)
	// The invalid pattern is skipped; the value is left unchanged.
	assert.Equal(t, map[string]any{}, got["somekey"])
}

// TestApplySchemaDefaults_PatternPropertySchemaNonMap covers the defensive
// continue in applySchemaDefaults when a patternProperties entry value is not a
// map[string]any.  The JSON Schema compiler rejects such schemas at build time,
// so this path is only reachable by calling the function directly.
func TestApplySchemaDefaults_PatternPropertySchemaNonMap(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"patternProperties": map[string]any{
			"^foo": "not-a-map", // non-map schema → must be silently skipped
			"^bar": map[string]any{
				"properties": map[string]any{
					"count": map[string]any{"default": 42},
				},
			},
		},
	}

	data := map[string]any{
		"fooKey": map[string]any{"x": 1},
		"barKey": map[string]any{},
	}
	result := applySchemaDefaults(data, schema)

	// The non-map pattern is silently skipped; fooKey is unchanged.
	fooVal, ok := result["fooKey"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, fooVal["x"])
	assert.NotContains(t, fooVal, "count")

	// The valid pattern still applies its default.
	barVal, ok := result["barKey"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 42, barVal["count"])
}

// TestApplyItemDefaults_NoItemsKey covers the early return in applyItemDefaults
// when the property schema has no "items" key.
func TestApplyItemDefaults_NoItemsKey(t *testing.T) {
	t.Parallel()

	propSchema := map[string]any{"type": "array"} // no "items" key
	slice := []any{"a", "b", "c"}
	got := applyItemDefaults(slice, propSchema)
	assert.Equal(t, slice, got)
}

func TestSchemaCanonicalization_RenamesValuesKeyToSchemaCase(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"apiVersion": map[string]any{"type": "string"},
		},
	}
	values := map[string]any{"apiversion": "v1"}
	got := canonicalizeSchemaKeys(values, schema)
	_, hasOld := got["apiversion"]
	assert.False(t, hasOld, "old lowercase key must be removed")
	assert.Equal(t, "v1", got["apiVersion"])
}

func TestSchemaCanonicalization_LeavesMatchingKeyAlone(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"apiVersion": map[string]any{"type": "string"},
		},
	}
	values := map[string]any{"apiVersion": "v2"}
	got := canonicalizeSchemaKeys(values, schema)
	assert.Equal(t, "v2", got["apiVersion"])
	assert.Len(t, got, 1)
}

func TestSchemaCanonicalization_LeavesUnknownKeysAlone(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"known": map[string]any{"type": "string"},
		},
	}
	values := map[string]any{
		"known":   "yes",
		"Unknown": "stays",
	}
	got := canonicalizeSchemaKeys(values, schema)
	assert.Equal(t, "yes", got["known"])
	assert.Equal(t, "stays", got["Unknown"])
}

func TestSchemaCanonicalization_NestedObject(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"server": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"Host": map[string]any{"type": "string"},
				},
			},
		},
	}
	values := map[string]any{
		"server": map[string]any{"host": "localhost"},
	}
	got := canonicalizeSchemaKeys(values, schema)
	nested, ok := got["server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", nested["Host"])
	_, hasOld := nested["host"]
	assert.False(t, hasOld)
}

func TestSchemaCanonicalization_ArrayItems(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"servers": map[string]any{
				"type": "array",
				"items": map[string]any{
					"properties": map[string]any{
						"Host": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	values := map[string]any{
		"servers": []any{
			map[string]any{"host": "a"},
			map[string]any{"HOST": "b"},
		},
	}
	got := canonicalizeSchemaKeys(values, schema)
	items, ok := got["servers"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	second, ok := items[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "a", first["Host"])
	assert.Equal(t, "b", second["Host"])
}

func TestSchemaCanonicalization_PatternPropertiesNotRenamed(t *testing.T) {
	t.Parallel()

	// patternProperties are dynamic; only "properties" keys get canonicalized.
	schema := map[string]any{
		"patternProperties": map[string]any{
			"^x-": map[string]any{"type": "string"},
		},
	}
	values := map[string]any{
		"x-Custom": "value",
		"X-Custom": "other",
	}
	got := canonicalizeSchemaKeys(values, schema)
	// Both keys survive unchanged because canonicalize only looks at "properties".
	assert.Equal(t, "value", got["x-Custom"])
	assert.Equal(t, "other", got["X-Custom"])
}

func TestApplySchemaDefaults_NoLongerLowercases(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"properties": map[string]any{
			"apiVersion": map[string]any{
				"type":    "string",
				"default": "v1",
			},
		},
	}
	values := map[string]any{}
	got := applySchemaDefaults(values, schema)
	// Default must be stored under the schema's declared casing, not lowercased.
	assert.Equal(t, "v1", got["apiVersion"])
	_, hasLower := got["apiversion"]
	assert.False(t, hasLower, "lowercase variant must not be added")
}
