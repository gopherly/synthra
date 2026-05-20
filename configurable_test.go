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
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurable_Get_ExactMatch(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"port": 8080})
	assert.Equal(t, 8080, v.Get("port"))
}

func TestConfigurable_Get_FoldMatch(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"apiVersion": "v1"})
	assert.Equal(t, "v1", v.Get("apiversion"))
	assert.Equal(t, "v1", v.Get("APIVERSION"))
	assert.Equal(t, "v1", v.Get("ApiVersion"))
}

func TestConfigurable_Get_NestedExact(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{
		"server": map[string]any{"port": 9090},
	})
	assert.Equal(t, 9090, v.Get("server.port"))
}

func TestConfigurable_Get_NestedFoldMatch(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{
		"Server": map[string]any{"Port": 9090},
	})
	assert.Equal(t, 9090, v.Get("server.port"))
}

func TestConfigurable_Get_MissingKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"foo": "bar"})
	assert.Nil(t, v.Get("missing"))
	assert.Nil(t, v.Get("foo.missing"))
	assert.Nil(t, v.Get("missing.nested"))
}

func TestConfigurable_Get_LiteralDotKey(t *testing.T) {
	t.Parallel()
	// A key that literally contains a dot takes precedence over traversal.
	v := newConfigurable(map[string]any{"a.b": "literal"})
	assert.Equal(t, "literal", v.Get("a.b"))
}

func TestConfigurable_Get_EmptyValues(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Nil(t, v.Get("anything"))
}

func TestConfigurable_Has_ExistingKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"host": "localhost"})
	assert.True(t, v.Has("host"))
	assert.True(t, v.Has("HOST"))
}

func TestConfigurable_Has_MissingKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"host": "localhost"})
	assert.False(t, v.Has("port"))
}

func TestConfigurable_Has_NestedKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{
		"db": map[string]any{"name": "prod"},
	})
	assert.True(t, v.Has("db.name"))
	assert.True(t, v.Has("DB.NAME"))
	assert.False(t, v.Has("db.missing"))
}

func TestConfigurable_Set_TopLevelNew(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	require.NoError(t, v.Set("port", 8080))
	assert.Equal(t, 8080, v.Get("port"))
}

func TestConfigurable_Set_TopLevelOverwrite(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"port": 8080})
	require.NoError(t, v.Set("PORT", 9090))
	// Key casing is preserved as-is for the matching key.
	assert.Equal(t, 9090, v.Get("port"))
}

func TestConfigurable_Set_CreatesIntermediateMaps(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	require.NoError(t, v.Set("server.port", 1234))
	assert.Equal(t, 1234, v.Get("server.port"))
}

func TestConfigurable_Set_ErrorOnNonMapIntermediate(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"server": "not-a-map"})
	err := v.Set("server.port", 8080)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server")
}

func TestConfigurable_Set_ThenGetViaDifferentCase(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	require.NoError(t, v.Set("LogLevel", "info"))
	assert.Equal(t, "info", v.Get("loglevel"))
	assert.Equal(t, "info", v.Get("LOGLEVEL"))
}

func TestConfigurable_Delete_MissingKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"foo": "bar"})
	assert.False(t, v.Delete("missing"))
}

func TestConfigurable_Delete_ExistingKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"foo": "bar"})
	assert.True(t, v.Delete("foo"))
	assert.Nil(t, v.Get("foo"))
}

func TestConfigurable_Delete_FoldMatch(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"apiVersion": "v1"})
	assert.True(t, v.Delete("apiversion"))
	assert.Nil(t, v.Get("apiVersion"))
}

func TestConfigurable_Delete_NestedKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{
		"db": map[string]any{"host": "localhost", "port": 5432},
	})
	assert.True(t, v.Delete("db.port"))
	assert.Nil(t, v.Get("db.port"))
	assert.Equal(t, "localhost", v.Get("db.host"))
}

func TestConfigurable_Keys_TopLevel(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"apiVersion": "v1", "port": 8080})
	keys := v.Keys()
	sort.Strings(keys)
	assert.Equal(t, []string{"apiVersion", "port"}, keys)
}

func TestConfigurable_Keys_Empty(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Empty(t, v.Keys())
}

func TestConfigurable_Walk_VisitsAllNodes(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{
		"a": map[string]any{
			"b": 1,
			"c": []any{"x", "y"},
		},
	})

	var visited []string
	v.Walk(func(path string, _ any) (any, bool) {
		visited = append(visited, path)
		return nil, false
	})

	sort.Strings(visited)
	assert.Contains(t, visited, "a")
	assert.Contains(t, visited, "a.b")
	assert.Contains(t, visited, "a.c")
	assert.Contains(t, visited, "a.c[0]")
	assert.Contains(t, visited, "a.c[1]")
}

func TestConfigurable_Walk_ReplacesValue(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"level": "DEBUG"})
	v.Walk(func(_ string, val any) (any, bool) {
		if s, ok := val.(string); ok {
			return s + "-replaced", true
		}
		return nil, false
	})
	assert.Equal(t, "DEBUG-replaced", v.Get("level"))
}

func TestConfigurable_Walk_SliceIndexFormat(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{
		"items": []any{
			map[string]any{"name": "first"},
			map[string]any{"name": "second"},
		},
	})

	var nestedPaths []string
	v.Walk(func(path string, _ any) (any, bool) {
		nestedPaths = append(nestedPaths, path)
		return nil, false
	})

	assert.Contains(t, nestedPaths, "items[0]")
	assert.Contains(t, nestedPaths, "items[0].name")
	assert.Contains(t, nestedPaths, "items[1]")
	assert.Contains(t, nestedPaths, "items[1].name")
}

func TestConfigurable_Raw_ReturnsSameMap(t *testing.T) {
	t.Parallel()
	m := map[string]any{"x": 1}
	v := newConfigurable(m)
	assert.Equal(t, m, v.Raw())
}

func TestConfigurable_Raw_MutationsVisibleThroughValues(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"a": 1})
	v.Raw()["b"] = 2
	assert.Equal(t, 2, v.Get("b"))
}

func TestConfigurable_Set_VisibleThroughRaw(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	require.NoError(t, v.Set("key", "val"))
	assert.Equal(t, "val", v.Raw()["key"])
}

func TestConfigurable_String_ExistingKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"msg": "hello"})
	got, err := v.String("msg")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestConfigurable_String_MissingKey(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.String("missing")
	require.Error(t, err)
}

func TestConfigurable_StringOr_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Equal(t, "default", v.StringOr("missing", "default"))
}

func TestConfigurable_Int_ParseString(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"port": "8080"})
	got, err := v.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, got)
}

func TestConfigurable_Int_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.Int("missing")
	require.Error(t, err)
}

func TestConfigurable_IntOr_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Equal(t, 9090, v.IntOr("missing", 9090))
}

func TestConfigurable_Int64_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"big": int64(1e12)})
	got, err := v.Int64("big")
	require.NoError(t, err)
	assert.Equal(t, int64(1e12), got)
}

func TestConfigurable_Float64_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"ratio": 0.5})
	got, err := v.Float64("ratio")
	require.NoError(t, err)
	assert.InDelta(t, 0.5, got, 0.0001)
}

func TestConfigurable_Bool_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"debug": true})
	got, err := v.Bool("debug")
	require.NoError(t, err)
	assert.True(t, got)
}

func TestConfigurable_BoolOr_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.True(t, v.BoolOr("missing", true))
}

func TestConfigurable_Duration_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"timeout": "30s"})
	got, err := v.Duration("timeout")
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, got)
}

func TestConfigurable_DurationOr_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Equal(t, time.Minute, v.DurationOr("missing", time.Minute))
}

func TestConfigurable_Time_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"ts": "2023-01-01T12:00:00Z"})
	got, err := v.Time("ts")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), got)
}

func TestConfigurable_StringSlice_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"tags": []any{"a", "b", "c"}})
	got, err := v.StringSlice("tags")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestConfigurable_IntSlice_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"ports": []any{80, 443, 8080}})
	got, err := v.IntSlice("ports")
	require.NoError(t, err)
	assert.Equal(t, []int{80, 443, 8080}, got)
}

func TestConfigurable_StringMap_Value(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"labels": map[string]any{"env": "prod", "region": "eu"}})
	got, err := v.StringMapString("labels")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"env": "prod", "region": "eu"}, got)
}

func TestConfigurable_TypedAccessors_WrongType(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"nested": map[string]any{"a": 1}})

	_, err := v.Int("nested")
	require.Error(t, err)

	_, err = v.Bool("nested")
	require.Error(t, err)
}

func TestConfigurable_IntOr_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"port": 3000})
	assert.Equal(t, 3000, v.IntOr("port", 9090))
}

func TestConfigurable_Int64_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.Int64("missing")
	require.Error(t, err)
}

func TestConfigurable_Int64Or_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"n": int64(42)})
	assert.Equal(t, int64(42), v.Int64Or("n", 0))
}

func TestConfigurable_Int64Or_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Equal(t, int64(99), v.Int64Or("missing", 99))
}

func TestConfigurable_Float64_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.Float64("missing")
	require.Error(t, err)
}

func TestConfigurable_Float64Or_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"r": 0.25})
	assert.InDelta(t, 0.25, v.Float64Or("r", 1.0), 0.0001)
}

func TestConfigurable_Float64Or_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.InDelta(t, 1.5, v.Float64Or("missing", 1.5), 0.0001)
}

func TestConfigurable_Bool_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.Bool("missing")
	require.Error(t, err)
}

func TestConfigurable_BoolOr_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"flag": false})
	assert.False(t, v.BoolOr("flag", true))
}

func TestConfigurable_Duration_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.Duration("missing")
	require.Error(t, err)
}

func TestConfigurable_DurationOr_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"ttl": "2m"})
	assert.Equal(t, 2*time.Minute, v.DurationOr("ttl", time.Hour))
}

func TestConfigurable_Time_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.Time("missing")
	require.Error(t, err)
}

func TestConfigurable_TimeOr_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"ts": "2023-06-15T10:00:00Z"})
	def := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	got := v.TimeOr("ts", def)
	assert.Equal(t, 2023, got.Year())
}

func TestConfigurable_TimeOr_Default(t *testing.T) {
	t.Parallel()
	def := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	v := newConfigurable(nil)
	assert.Equal(t, def, v.TimeOr("missing", def))
}

func TestConfigurable_StringSlice_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.StringSlice("missing")
	require.Error(t, err)
}

func TestConfigurable_StringSliceOr_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"tags": []any{"x", "y"}})
	assert.Equal(t, []string{"x", "y"}, v.StringSliceOr("tags", nil))
}

func TestConfigurable_StringSliceOr_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Equal(t, []string{"a"}, v.StringSliceOr("missing", []string{"a"}))
}

func TestConfigurable_IntSlice_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.IntSlice("missing")
	require.Error(t, err)
}

func TestConfigurable_IntSliceOr_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"ports": []any{80, 443}})
	assert.Equal(t, []int{80, 443}, v.IntSliceOr("ports", nil))
}

func TestConfigurable_IntSliceOr_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	assert.Equal(t, []int{1, 2}, v.IntSliceOr("missing", []int{1, 2}))
}

func TestConfigurable_StringMap_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"labels": map[string]any{"env": "prod"}})
	got, err := v.StringMap("labels")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"env": "prod"}, got)
}

func TestConfigurable_StringMap_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.StringMap("missing")
	require.Error(t, err)
}

func TestConfigurable_StringMapOr_Found(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"m": map[string]any{"a": "b"}})
	assert.Equal(t, map[string]any{"a": "b"}, v.StringMapOr("m", nil))
}

func TestConfigurable_StringMapOr_Default(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	def := map[string]any{"x": "y"}
	assert.Equal(t, def, v.StringMapOr("missing", def))
}

func TestConfigurable_StringMapString_Missing(t *testing.T) {
	t.Parallel()
	v := newConfigurable(nil)
	_, err := v.StringMapString("missing")
	require.Error(t, err)
}

func TestConfigurable_Has_LiteralDotKey(t *testing.T) {
	t.Parallel()
	// A key that literally contains a dot must be found by Has, exercising the
	// top-level findKeyFold hit before dot-traversal is attempted.
	v := newConfigurable(map[string]any{"a.b": "literal"})
	assert.True(t, v.Has("a.b"))
}

func TestConfigurable_Has_NestedMissingIntermediate(t *testing.T) {
	t.Parallel()
	// Multi-segment path where the first intermediate segment does not exist
	// at all, exercising the k=="" early-return inside the traversal loop.
	v := newConfigurable(map[string]any{"other": "val"})
	assert.False(t, v.Has("nonexistent.sub"))
}

func TestConfigurable_Has_NestedNonMapIntermediate(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"a": "scalar-not-map"})
	assert.False(t, v.Has("a.b"))
}

func TestConfigurable_Delete_NestedMissingIntermediate(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"other": "val"})
	assert.False(t, v.Delete("missing.key"))
}

func TestConfigurable_Delete_NestedMissingFinalKey(t *testing.T) {
	t.Parallel()
	// Intermediate map exists but the final key does not.
	v := newConfigurable(map[string]any{"db": map[string]any{"host": "localhost"}})
	assert.False(t, v.Delete("db.nonexistent"))
}

func TestConfigurable_Delete_NestedNonMapIntermediate(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"a": "scalar"})
	assert.False(t, v.Delete("a.b"))
}

func TestConfigurable_Walk_ReplacesSliceElement(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{"tags": []any{"old"}})
	v.Walk(func(path string, val any) (any, bool) {
		if path == "tags[0]" {
			return "new", true
		}
		return nil, false
	})
	got, err := v.StringSlice("tags")
	require.NoError(t, err)
	assert.Equal(t, []string{"new"}, got)
}

func TestConfigurable_Walk_NestedSliceInSlice(t *testing.T) {
	t.Parallel()
	v := newConfigurable(map[string]any{
		"matrix": []any{
			[]any{1, 2},
			[]any{3, 4},
		},
	})
	var paths []string
	v.Walk(func(path string, _ any) (any, bool) {
		paths = append(paths, path)
		return nil, false
	})
	assert.Contains(t, paths, "matrix[0]")
	assert.Contains(t, paths, "matrix[0][0]")
	assert.Contains(t, paths, "matrix[1][1]")
}
