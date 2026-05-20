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

package synthra

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers to build test fixtures

func envSlice(envs ...map[string]any) []any {
	out := make([]any, 0, len(envs))
	for _, e := range envs {
		out = append(out, e)
	}
	return out
}

func mixedSlice() []any {
	return []any{
		map[string]any{"name": "first"},
		"not-a-map",
		42,
		map[string]any{"name": "last"},
	}
}

func TestSliceLen_MissingPath(t *testing.T) {
	c := &Configuration{m: map[string]any{}}
	assert.Equal(t, 0, c.SliceLen("environments"))
}

func TestSliceLen_NonSlice(t *testing.T) {
	c := &Configuration{m: map[string]any{"environments": "not-a-slice"}}
	assert.Equal(t, 0, c.SliceLen("environments"))
}

func TestSliceLen_EmptySlice(t *testing.T) {
	c := &Configuration{m: map[string]any{"environments": []any{}}}
	assert.Equal(t, 0, c.SliceLen("environments"))
}

func TestSliceLen_NonEmpty(t *testing.T) {
	c := &Configuration{m: map[string]any{
		"environments": envSlice(
			map[string]any{"name": "dev"},
			map[string]any{"name": "prod"},
		),
	}}
	assert.Equal(t, 2, c.SliceLen("environments"))
}

func TestSliceLen_NilReceiver(t *testing.T) {
	var c *Configuration
	assert.Equal(t, 0, c.SliceLen("environments"))
}

func TestEachMap_MissingPath(t *testing.T) {
	c := &Configuration{m: map[string]any{}}
	var count int
	for range c.EachMap("environments") {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestEachMap_NonSlice(t *testing.T) {
	c := &Configuration{m: map[string]any{"environments": "scalar"}}
	var count int
	for range c.EachMap("environments") {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestEachMap_EmptySlice(t *testing.T) {
	c := &Configuration{m: map[string]any{"environments": []any{}}}
	var count int
	for range c.EachMap("environments") {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestEachMap_AllMapElements(t *testing.T) {
	c := &Configuration{m: map[string]any{
		"environments": envSlice(
			map[string]any{"name": "dev"},
			map[string]any{"name": "prod"},
		),
	}}
	var names []string
	for _, e := range c.EachMap("environments") {
		names = append(names, e.StringOr("name", ""))
	}
	assert.Equal(t, []string{"dev", "prod"}, names)
}

func TestEachMap_MixedTypeSlice_SkipsNonMaps(t *testing.T) {
	c := &Configuration{m: map[string]any{"items": mixedSlice()}}
	var names []string
	for _, e := range c.EachMap("items") {
		names = append(names, e.StringOr("name", ""))
	}
	// only map elements yielded
	assert.Equal(t, []string{"first", "last"}, names)
}

func TestEachMap_NilReceiver(t *testing.T) {
	var c *Configuration
	var count int
	for range c.EachMap("environments") {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestFind_MissingPath(t *testing.T) {
	c := &Configuration{m: map[string]any{}}
	assert.Nil(t, c.Find("environments", "name", "prod"))
}

func TestFind_NoMatch(t *testing.T) {
	c := &Configuration{m: map[string]any{
		"environments": envSlice(map[string]any{"name": "dev"}),
	}}
	assert.Nil(t, c.Find("environments", "name", "prod"))
}

func TestFind_Match(t *testing.T) {
	c := &Configuration{m: map[string]any{
		"environments": envSlice(
			map[string]any{"name": "dev", "port": 8080},
			map[string]any{"name": "prod", "port": 443},
		),
	}}
	got := c.Find("environments", "name", "prod")
	require.NotNil(t, got)
	assert.Equal(t, 443, got.IntOr("port", 0))
}

func TestFind_CaseInsensitiveField(t *testing.T) {
	c := &Configuration{m: map[string]any{
		"environments": envSlice(map[string]any{"Name": "Prod"}),
	}}
	got := c.Find("environments", "name", "Prod")
	require.NotNil(t, got)
	assert.Equal(t, "Prod", got.StringOr("Name", ""))
}

func TestFind_NilReceiver(t *testing.T) {
	var c *Configuration
	assert.Nil(t, c.Find("environments", "name", "prod"))
}

func TestFindFunc_PredicateShortCircuit(t *testing.T) {
	visited := 0
	c := &Configuration{m: map[string]any{
		"environments": envSlice(
			map[string]any{"name": "dev"},
			map[string]any{"name": "staging"},
			map[string]any{"name": "prod"},
		),
	}}
	got := c.FindFunc("environments", func(e *Configuration) bool {
		visited++
		return e.StringOr("name", "") == "staging"
	})
	require.NotNil(t, got)
	// should stop after visiting dev and staging (2 visits)
	assert.Equal(t, 2, visited)
	assert.Equal(t, "staging", got.StringOr("name", ""))
}

func TestFindFunc_NilReceiver(t *testing.T) {
	var c *Configuration
	got := c.FindFunc("environments", func(_ *Configuration) bool { return true })
	assert.Nil(t, got)
}

func TestConfigurableEachMap_YieldsMutableWrappers(t *testing.T) {
	shared := []any{
		map[string]any{"name": "dev", "port": 8080},
	}
	c := newConfigurable(map[string]any{"environments": shared})

	for _, e := range c.EachMap("environments") {
		require.NoError(t, e.Set("port", 9090))
	}

	// mutation reaches back into parent's backing slice
	slice, ok := c.m["environments"].([]any)
	require.True(t, ok)
	entry, ok := slice[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 9090, entry["port"])
}

func TestConfigurableEachMap_MixedTypeSlice_SkipsNonMaps(t *testing.T) {
	c := newConfigurable(map[string]any{"items": mixedSlice()})
	var names []string
	for _, e := range c.EachMap("items") {
		names = append(names, e.StringOr("name", ""))
	}
	assert.Equal(t, []string{"first", "last"}, names)
}

func TestConfigurableFind_MutationVisible(t *testing.T) {
	parentMap := map[string]any{
		"environments": []any{
			map[string]any{"name": "prod", "port": 443},
		},
	}
	c := newConfigurable(parentMap)

	env := c.Find("environments", "name", "prod")
	require.NotNil(t, env)
	require.NoError(t, env.Set("port", 8443))

	// mutation is reflected in the parent's backing map
	slice, ok := parentMap["environments"].([]any)
	require.True(t, ok)
	entry, ok := slice[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 8443, entry["port"])
}

func TestConfigurableFind_NilReceiver(t *testing.T) {
	var c *Configurable
	assert.Nil(t, c.Find("environments", "name", "prod"))
}

func TestConfigurableFindFunc_PredicateShortCircuit(t *testing.T) {
	visited := 0
	c := newConfigurable(map[string]any{
		"environments": envSlice(
			map[string]any{"name": "dev"},
			map[string]any{"name": "prod"},
		),
	})
	got := c.FindFunc("environments", func(e *Configurable) bool {
		visited++
		return e.StringOr("name", "") == "dev"
	})
	require.NotNil(t, got)
	assert.Equal(t, 1, visited)
}
