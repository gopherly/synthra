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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindKeyFold_EmptyMap(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", findKeyFold(map[string]any{}, "foo"))
}

func TestFindKeyFold_NilMap(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", findKeyFold(nil, "foo"))
}

func TestFindKeyFold_ExactMatch(t *testing.T) {
	t.Parallel()
	m := map[string]any{"apiVersion": "v1"}
	assert.Equal(t, "apiVersion", findKeyFold(m, "apiVersion"))
}

func TestFindKeyFold_FoldMatch(t *testing.T) {
	t.Parallel()
	m := map[string]any{"apiVersion": "v1"}
	assert.Equal(t, "apiVersion", findKeyFold(m, "apiversion"))
	assert.Equal(t, "apiVersion", findKeyFold(m, "APIVERSION"))
	assert.Equal(t, "apiVersion", findKeyFold(m, "ApiVersion"))
}

func TestFindKeyFold_NoMatch(t *testing.T) {
	t.Parallel()
	m := map[string]any{"port": 8080}
	assert.Equal(t, "", findKeyFold(m, "host"))
}

func TestFindKeyFold_MultiKeyMap_ExactPrecedesOthers(t *testing.T) {
	t.Parallel()
	// Both "foo" and "FOO" present: exact match on "foo" must win over fold.
	m := map[string]any{"foo": "lower", "FOO": "upper"}
	assert.Equal(t, "foo", findKeyFold(m, "foo"))
	assert.Equal(t, "FOO", findKeyFold(m, "FOO"))
}

func TestFindKeyFold_MultiKeyMapFoldOnly(t *testing.T) {
	t.Parallel()
	m := map[string]any{"ApiVersion": "v1", "Port": 9090}
	got := findKeyFold(m, "apiversion")
	assert.Equal(t, "ApiVersion", got)
}

// --- canonicalizeSchemaKeys ---

func TestCanonicalizeSchemaKeys_NilValues(t *testing.T) {
	t.Parallel()
	got := canonicalizeSchemaKeys(nil, map[string]any{
		"properties": map[string]any{"name": map[string]any{}},
	})
	assert.Nil(t, got)
}

func TestCanonicalizeSchemaKeys_SchemaWithoutProperties(t *testing.T) {
	t.Parallel()
	values := map[string]any{"Name": "bob"}
	schema := map[string]any{"type": "object"}
	got := canonicalizeSchemaKeys(values, schema)
	// No properties in schema: values unchanged.
	assert.Equal(t, map[string]any{"Name": "bob"}, got)
}

func TestCanonicalizeSchemaKeys_RenamesCaseDifferentKey(t *testing.T) {
	t.Parallel()
	// Schema declares "apiVersion"; values has "apiversion" (lowercase).
	schema := map[string]any{
		"properties": map[string]any{
			"apiVersion": map[string]any{"type": "string"},
		},
	}
	values := map[string]any{"apiversion": "v1"}
	got := canonicalizeSchemaKeys(values, schema)
	assert.Equal(t, "v1", got["apiVersion"])
	assert.NotContains(t, got, "apiversion")
}

func TestCanonicalizeSchemaKeys_LeavesMatchingKeyAlone(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"properties": map[string]any{
			"apiVersion": map[string]any{"type": "string"},
		},
	}
	values := map[string]any{"apiVersion": "v2"}
	got := canonicalizeSchemaKeys(values, schema)
	assert.Equal(t, "v2", got["apiVersion"])
}

func TestCanonicalizeSchemaKeys_LeavesUnknownKeysAlone(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}
	values := map[string]any{"name": "bob", "EXTRA": "keep"}
	got := canonicalizeSchemaKeys(values, schema)
	assert.Equal(t, "bob", got["name"])
	assert.Equal(t, "keep", got["EXTRA"])
}

func TestCanonicalizeSchemaKeys_RecursesIntoNestedObject(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"properties": map[string]any{
			"server": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"Port": map[string]any{"type": "integer"},
				},
			},
		},
	}
	values := map[string]any{
		"server": map[string]any{"port": 8080},
	}
	got := canonicalizeSchemaKeys(values, schema)
	srv, ok := got["server"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, 8080, srv["Port"])
	assert.NotContains(t, srv, "port")
}

func TestCanonicalizeSchemaKeys_RecursesIntoArrayItems(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"properties": map[string]any{
			"envs": map[string]any{
				"type": "array",
				"items": map[string]any{
					"properties": map[string]any{
						"EnvFile": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	values := map[string]any{
		"envs": []any{
			map[string]any{"envfile": ".env.prod"},
			map[string]any{"envfile": ".env.staging"},
		},
	}
	got := canonicalizeSchemaKeys(values, schema)
	items, ok := got["envs"].([]any)
	assert.True(t, ok)
	assert.Len(t, items, 2)
	for _, elem := range items {
		m, elemOK := elem.(map[string]any)
		assert.True(t, elemOK)
		assert.Contains(t, m, "EnvFile")
		assert.NotContains(t, m, "envfile")
	}
}

func TestCanonicalizeSchemaKeys_NonMapArrayItemsSkipped(t *testing.T) {
	t.Parallel()
	schema := map[string]any{
		"properties": map[string]any{
			"tags": map[string]any{
				"type": "array",
				"items": map[string]any{
					"properties": map[string]any{
						"Name": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	// Items are plain strings, not maps — must not panic.
	values := map[string]any{
		"tags": []any{"alpha", "beta"},
	}
	got := canonicalizeSchemaKeys(values, schema)
	assert.Equal(t, []any{"alpha", "beta"}, got["tags"])
}
