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

func conf(m map[string]any) *Configuration { return &Configuration{m: m} }

func TestConfiguration_Get_ExactMatch(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"port": 8080})
	assert.Equal(t, 8080, s.Get("port"))
}

func TestConfiguration_Get_FoldMatch(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"apiVersion": "v1"})
	assert.Equal(t, "v1", s.Get("apiversion"))
	assert.Equal(t, "v1", s.Get("APIVERSION"))
}

func TestConfiguration_Get_Nested(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"db": map[string]any{"host": "localhost"}})
	assert.Equal(t, "localhost", s.Get("db.host"))
}

func TestConfiguration_Get_Missing(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"a": 1})
	assert.Nil(t, s.Get("missing"))
	assert.Nil(t, s.Get("a.missing"))
}

func TestConfiguration_Has_ExistingKey(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"host": "localhost"})
	assert.True(t, s.Has("host"))
	assert.True(t, s.Has("HOST"))
}

func TestConfiguration_Has_NestedKey(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"db": map[string]any{"port": 5432}})
	assert.True(t, s.Has("db.port"))
	assert.False(t, s.Has("db.missing"))
}

func TestConfiguration_Has_NestedNonMapIntermediate(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"a": "scalar"})
	assert.False(t, s.Has("a.b"))
}

func TestConfiguration_Has_MissingKey(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"host": "localhost"})
	assert.False(t, s.Has("missing"))
}

func TestConfiguration_Keys(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"b": 2, "a": 1})
	keys := s.Keys()
	sort.Strings(keys)
	assert.Equal(t, []string{"a", "b"}, keys)
}

func TestConfiguration_Keys_Empty(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.Empty(t, s.Keys())
}

func TestConfiguration_Raw_IsClone(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"x": 1})
	raw := s.Raw()
	assert.Equal(t, 1, raw["x"])
	raw["x"] = 99
	assert.Equal(t, 1, s.Get("x"), "Raw must return a clone; mutation must not affect Snapshot")
}

// --- String ---

func TestConfiguration_String_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"msg": "hello"})
	got, err := s.String("msg")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestConfiguration_String_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.String("nope")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_StringOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"msg": "hello"})
	assert.Equal(t, "hello", s.StringOr("msg", "default"))
}

func TestConfiguration_StringOr_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.Equal(t, "default", s.StringOr("missing", "default"))
}

// --- Int ---

func TestConfiguration_Int_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"port": 8080})
	got, err := s.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, got)
}

func TestConfiguration_Int_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.Int("port")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_IntOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"port": 8080})
	assert.Equal(t, 8080, s.IntOr("port", 9090))
}

func TestConfiguration_IntOr_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.Equal(t, 9090, s.IntOr("missing", 9090))
}

// --- Int64 ---

func TestConfiguration_Int64_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"big": int64(1e12)})
	got, err := s.Int64("big")
	require.NoError(t, err)
	assert.Equal(t, int64(1e12), got)
}

func TestConfiguration_Int64_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.Int64("big")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_Int64Or_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"n": int64(42)})
	assert.Equal(t, int64(42), s.Int64Or("n", 0))
}

func TestConfiguration_Int64Or_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.Equal(t, int64(99), s.Int64Or("missing", 99))
}

// --- Float64 ---

func TestConfiguration_Float64_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"rate": 0.75})
	got, err := s.Float64("rate")
	require.NoError(t, err)
	assert.InDelta(t, 0.75, got, 0.0001)
}

func TestConfiguration_Float64_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.Float64("rate")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_Float64Or_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"rate": 0.5})
	assert.InDelta(t, 0.5, s.Float64Or("rate", 1.0), 0.0001)
}

func TestConfiguration_Float64Or_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.InDelta(t, 1.0, s.Float64Or("missing", 1.0), 0.0001)
}

// --- Bool ---

func TestConfiguration_Bool_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"debug": true})
	got, err := s.Bool("debug")
	require.NoError(t, err)
	assert.True(t, got)
}

func TestConfiguration_Bool_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.Bool("debug")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_BoolOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"debug": false})
	assert.False(t, s.BoolOr("debug", true))
}

func TestConfiguration_BoolOr_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.True(t, s.BoolOr("missing", true))
}

// --- Duration ---

func TestConfiguration_Duration_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"timeout": "5s"})
	got, err := s.Duration("timeout")
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, got)
}

func TestConfiguration_Duration_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.Duration("timeout")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_DurationOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"timeout": "10s"})
	assert.Equal(t, 10*time.Second, s.DurationOr("timeout", time.Minute))
}

func TestConfiguration_DurationOr_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.Equal(t, time.Minute, s.DurationOr("missing", time.Minute))
}

// --- Time ---

func TestConfiguration_Time_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"ts": "2023-06-15T10:00:00Z"})
	got, err := s.Time("ts")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC), got)
}

func TestConfiguration_Time_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.Time("ts")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_TimeOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"ts": "2023-06-15T10:00:00Z"})
	def := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	got := s.TimeOr("ts", def)
	assert.Equal(t, 2023, got.Year())
}

func TestConfiguration_TimeOr_Default(t *testing.T) {
	t.Parallel()
	def := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	s := emptyConfiguration()
	assert.Equal(t, def, s.TimeOr("missing", def))
}

// --- StringSlice ---

func TestConfiguration_StringSlice_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"tags": []any{"a", "b"}})
	got, err := s.StringSlice("tags")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, got)
}

func TestConfiguration_StringSlice_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.StringSlice("tags")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_StringSliceOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"tags": []any{"x"}})
	assert.Equal(t, []string{"x"}, s.StringSliceOr("tags", nil))
}

func TestConfiguration_StringSliceOr_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.Equal(t, []string{"default"}, s.StringSliceOr("missing", []string{"default"}))
}

// --- IntSlice ---

func TestConfiguration_IntSlice_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"ports": []any{80, 443}})
	got, err := s.IntSlice("ports")
	require.NoError(t, err)
	assert.Equal(t, []int{80, 443}, got)
}

func TestConfiguration_IntSlice_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.IntSlice("ports")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_IntSliceOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"ports": []any{8080}})
	assert.Equal(t, []int{8080}, s.IntSliceOr("ports", nil))
}

func TestConfiguration_IntSliceOr_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	assert.Equal(t, []int{3000}, s.IntSliceOr("missing", []int{3000}))
}

// --- StringMap ---

func TestConfiguration_StringMap_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"labels": map[string]any{"env": "prod"}})
	got, err := s.StringMap("labels")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"env": "prod"}, got)
}

func TestConfiguration_StringMap_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.StringMap("labels")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_StringMapOr_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"m": map[string]any{"a": "b"}})
	assert.Equal(t, map[string]any{"a": "b"}, s.StringMapOr("m", nil))
}

func TestConfiguration_StringMapOr_Default(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	def := map[string]any{"x": "y"}
	assert.Equal(t, def, s.StringMapOr("missing", def))
}

// --- StringMapString ---

func TestConfiguration_StringMapString_Found(t *testing.T) {
	t.Parallel()
	s := conf(map[string]any{"meta": map[string]any{"k": "v"}})
	got, err := s.StringMapString("meta")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k": "v"}, got)
}

func TestConfiguration_StringMapString_Missing(t *testing.T) {
	t.Parallel()
	s := emptyConfiguration()
	_, err := s.StringMapString("meta")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestConfiguration_NilReceiver(t *testing.T) {
	t.Parallel()

	var s *Configuration
	path := "any.key"

	assert.Nil(t, s.Get(path))
	assert.False(t, s.Has(path))
	assert.Nil(t, s.Keys())
	assert.Nil(t, s.Raw())
	assert.Equal(t, 0, s.SliceLen(path))

	assert.Equal(t, "def", s.StringOr(path, "def"))
	assert.Equal(t, 1, s.IntOr(path, 1))
	assert.Equal(t, int64(2), s.Int64Or(path, 2))
	assert.InDelta(t, 3.0, s.Float64Or(path, 3.0), 0.0001)
	assert.True(t, s.BoolOr(path, true))
	assert.Equal(t, time.Second, s.DurationOr(path, time.Second))
	defTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, defTime, s.TimeOr(path, defTime))
	assert.Equal(t, []string{"a"}, s.StringSliceOr(path, []string{"a"}))
	assert.Equal(t, []int{4}, s.IntSliceOr(path, []int{4}))
	defMap := map[string]any{"k": "v"}
	assert.Equal(t, defMap, s.StringMapOr(path, defMap))

	_, err := s.String(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.Int(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.Int64(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.Float64(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.Bool(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.Duration(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.Time(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.StringSlice(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.IntSlice(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.StringMap(path)
	require.ErrorIs(t, err, ErrNilConfig)
	_, err = s.StringMapString(path)
	require.ErrorIs(t, err, ErrNilConfig)
}
