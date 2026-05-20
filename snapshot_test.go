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

func snap(m map[string]any) *Snapshot { return &Snapshot{m: m} }

func TestSnapshot_Get_ExactMatch(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"port": 8080})
	assert.Equal(t, 8080, s.Get("port"))
}

func TestSnapshot_Get_FoldMatch(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"apiVersion": "v1"})
	assert.Equal(t, "v1", s.Get("apiversion"))
	assert.Equal(t, "v1", s.Get("APIVERSION"))
}

func TestSnapshot_Get_Nested(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"db": map[string]any{"host": "localhost"}})
	assert.Equal(t, "localhost", s.Get("db.host"))
}

func TestSnapshot_Get_Missing(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"a": 1})
	assert.Nil(t, s.Get("missing"))
	assert.Nil(t, s.Get("a.missing"))
}

func TestSnapshot_Has_ExistingKey(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"host": "localhost"})
	assert.True(t, s.Has("host"))
	assert.True(t, s.Has("HOST"))
}

func TestSnapshot_Has_NestedKey(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"db": map[string]any{"port": 5432}})
	assert.True(t, s.Has("db.port"))
	assert.False(t, s.Has("db.missing"))
}

func TestSnapshot_Has_NestedNonMapIntermediate(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"a": "scalar"})
	assert.False(t, s.Has("a.b"))
}

func TestSnapshot_Has_MissingKey(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"host": "localhost"})
	assert.False(t, s.Has("missing"))
}

func TestSnapshot_Keys(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"b": 2, "a": 1})
	keys := s.Keys()
	sort.Strings(keys)
	assert.Equal(t, []string{"a", "b"}, keys)
}

func TestSnapshot_Keys_Empty(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.Empty(t, s.Keys())
}

func TestSnapshot_Raw_IsClone(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"x": 1})
	raw := s.Raw()
	assert.Equal(t, 1, raw["x"])
	raw["x"] = 99
	assert.Equal(t, 1, s.Get("x"), "Raw must return a clone; mutation must not affect Snapshot")
}

// --- String ---

func TestSnapshot_String_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"msg": "hello"})
	got, err := s.String("msg")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestSnapshot_String_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.String("nope")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_StringOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"msg": "hello"})
	assert.Equal(t, "hello", s.StringOr("msg", "default"))
}

func TestSnapshot_StringOr_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.Equal(t, "default", s.StringOr("missing", "default"))
}

// --- Int ---

func TestSnapshot_Int_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"port": 8080})
	got, err := s.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, got)
}

func TestSnapshot_Int_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.Int("port")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_IntOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"port": 8080})
	assert.Equal(t, 8080, s.IntOr("port", 9090))
}

func TestSnapshot_IntOr_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.Equal(t, 9090, s.IntOr("missing", 9090))
}

// --- Int64 ---

func TestSnapshot_Int64_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"big": int64(1e12)})
	got, err := s.Int64("big")
	require.NoError(t, err)
	assert.Equal(t, int64(1e12), got)
}

func TestSnapshot_Int64_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.Int64("big")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_Int64Or_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"n": int64(42)})
	assert.Equal(t, int64(42), s.Int64Or("n", 0))
}

func TestSnapshot_Int64Or_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.Equal(t, int64(99), s.Int64Or("missing", 99))
}

// --- Float64 ---

func TestSnapshot_Float64_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"rate": 0.75})
	got, err := s.Float64("rate")
	require.NoError(t, err)
	assert.InDelta(t, 0.75, got, 0.0001)
}

func TestSnapshot_Float64_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.Float64("rate")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_Float64Or_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"rate": 0.5})
	assert.InDelta(t, 0.5, s.Float64Or("rate", 1.0), 0.0001)
}

func TestSnapshot_Float64Or_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.InDelta(t, 1.0, s.Float64Or("missing", 1.0), 0.0001)
}

// --- Bool ---

func TestSnapshot_Bool_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"debug": true})
	got, err := s.Bool("debug")
	require.NoError(t, err)
	assert.True(t, got)
}

func TestSnapshot_Bool_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.Bool("debug")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_BoolOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"debug": false})
	assert.False(t, s.BoolOr("debug", true))
}

func TestSnapshot_BoolOr_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.True(t, s.BoolOr("missing", true))
}

// --- Duration ---

func TestSnapshot_Duration_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"timeout": "5s"})
	got, err := s.Duration("timeout")
	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, got)
}

func TestSnapshot_Duration_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.Duration("timeout")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_DurationOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"timeout": "10s"})
	assert.Equal(t, 10*time.Second, s.DurationOr("timeout", time.Minute))
}

func TestSnapshot_DurationOr_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.Equal(t, time.Minute, s.DurationOr("missing", time.Minute))
}

// --- Time ---

func TestSnapshot_Time_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"ts": "2023-06-15T10:00:00Z"})
	got, err := s.Time("ts")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC), got)
}

func TestSnapshot_Time_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.Time("ts")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_TimeOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"ts": "2023-06-15T10:00:00Z"})
	def := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	got := s.TimeOr("ts", def)
	assert.Equal(t, 2023, got.Year())
}

func TestSnapshot_TimeOr_Default(t *testing.T) {
	t.Parallel()
	def := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	s := emptySnapshot()
	assert.Equal(t, def, s.TimeOr("missing", def))
}

// --- StringSlice ---

func TestSnapshot_StringSlice_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"tags": []any{"a", "b"}})
	got, err := s.StringSlice("tags")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, got)
}

func TestSnapshot_StringSlice_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.StringSlice("tags")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_StringSliceOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"tags": []any{"x"}})
	assert.Equal(t, []string{"x"}, s.StringSliceOr("tags", nil))
}

func TestSnapshot_StringSliceOr_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.Equal(t, []string{"default"}, s.StringSliceOr("missing", []string{"default"}))
}

// --- IntSlice ---

func TestSnapshot_IntSlice_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"ports": []any{80, 443}})
	got, err := s.IntSlice("ports")
	require.NoError(t, err)
	assert.Equal(t, []int{80, 443}, got)
}

func TestSnapshot_IntSlice_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.IntSlice("ports")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_IntSliceOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"ports": []any{8080}})
	assert.Equal(t, []int{8080}, s.IntSliceOr("ports", nil))
}

func TestSnapshot_IntSliceOr_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	assert.Equal(t, []int{3000}, s.IntSliceOr("missing", []int{3000}))
}

// --- StringMap ---

func TestSnapshot_StringMap_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"labels": map[string]any{"env": "prod"}})
	got, err := s.StringMap("labels")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"env": "prod"}, got)
}

func TestSnapshot_StringMap_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.StringMap("labels")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestSnapshot_StringMapOr_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"m": map[string]any{"a": "b"}})
	assert.Equal(t, map[string]any{"a": "b"}, s.StringMapOr("m", nil))
}

func TestSnapshot_StringMapOr_Default(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	def := map[string]any{"x": "y"}
	assert.Equal(t, def, s.StringMapOr("missing", def))
}

// --- StringMapString ---

func TestSnapshot_StringMapString_Found(t *testing.T) {
	t.Parallel()
	s := snap(map[string]any{"meta": map[string]any{"k": "v"}})
	got, err := s.StringMapString("meta")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k": "v"}, got)
}

func TestSnapshot_StringMapString_Missing(t *testing.T) {
	t.Parallel()
	s := emptySnapshot()
	_, err := s.StringMapString("meta")
	require.ErrorIs(t, err, ErrKeyNotFound)
}
