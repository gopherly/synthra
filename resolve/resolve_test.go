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

package resolve_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopherly.dev/synthra/resolve"
)

func TestVars(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		r := resolve.Vars(map[string]string{"PORT": "8080", "HOST": "localhost"})
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})

	t.Run("not found", func(t *testing.T) {
		r := resolve.Vars(map[string]string{"PORT": "8080"})
		val, ok := r("MISSING")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("nil map", func(t *testing.T) {
		r := resolve.Vars(nil)
		val, ok := r("PORT")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("empty map", func(t *testing.T) {
		r := resolve.Vars(map[string]string{})
		val, ok := r("PORT")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("keys are case-sensitive", func(t *testing.T) {
		r := resolve.Vars(map[string]string{"port": "8080"})
		_, ok := r("PORT")
		assert.False(t, ok)
		val, ok := r("port")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})
}

func TestOS(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		t.Setenv("SYNTHRA_TEST_VAR", "hello")
		r := resolve.OS()
		val, ok := r("SYNTHRA_TEST_VAR")
		assert.True(t, ok)
		assert.Equal(t, "hello", val)
	})

	t.Run("not found", func(t *testing.T) {
		r := resolve.OS()
		val, ok := r("SYNTHRA_DEFINITELY_NOT_SET_XYZ123")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("reads live environment", func(t *testing.T) {
		r := resolve.OS()
		t.Setenv("SYNTHRA_LIVE_VAR", "first")
		val, ok := r("SYNTHRA_LIVE_VAR")
		assert.True(t, ok)
		assert.Equal(t, "first", val)

		t.Setenv("SYNTHRA_LIVE_VAR", "second")
		val, ok = r("SYNTHRA_LIVE_VAR")
		assert.True(t, ok)
		assert.Equal(t, "second", val)
	})
}

func TestOSPrefix(t *testing.T) {
	t.Run("found with prefix stripped", func(t *testing.T) {
		t.Setenv("APP_PORT", "9090")
		r := resolve.OSPrefix("APP_")
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "9090", val)
	})

	t.Run("not found when prefixed var not set", func(t *testing.T) {
		r := resolve.OSPrefix("APP_")
		val, ok := r("SYNTHRA_DEFINITELY_NOT_SET_XYZ123")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("unprefixed var is not found", func(t *testing.T) {
		t.Setenv("PORT", "3000")
		r := resolve.OSPrefix("APP_")
		_, ok := r("PORT")
		// resolve.OSPrefix("APP_") looks for APP_PORT, not PORT
		assert.False(t, ok)
	})

	t.Run("empty prefix acts like OS resolver", func(t *testing.T) {
		t.Setenv("SYNTHRA_EMPTY_PREFIX_VAR", "value")
		r := resolve.OSPrefix("")
		val, ok := r("SYNTHRA_EMPTY_PREFIX_VAR")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("reads live environment", func(t *testing.T) {
		t.Setenv("DPY_VAR_HOST", "db.local")
		r := resolve.OSPrefix("DPY_VAR_")
		val, ok := r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "db.local", val)
	})
}

func TestChain(t *testing.T) {
	t.Run("last resolver wins", func(t *testing.T) {
		r := resolve.Chain(
			resolve.Vars(map[string]string{"PORT": "3000"}),
			resolve.Vars(map[string]string{"PORT": "8080"}),
		)
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})

	t.Run("falls back to earlier resolver", func(t *testing.T) {
		r := resolve.Chain(
			resolve.Vars(map[string]string{"PORT": "3000"}),
			resolve.Vars(map[string]string{"HOST": "localhost"}),
		)
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "3000", val)
	})

	t.Run("not found when no resolver has the key", func(t *testing.T) {
		r := resolve.Chain(
			resolve.Vars(map[string]string{"PORT": "3000"}),
			resolve.Vars(map[string]string{"HOST": "localhost"}),
		)
		val, ok := r("MISSING")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("no resolvers returns not found", func(t *testing.T) {
		r := resolve.Chain()
		val, ok := r("PORT")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("nil resolver in chain is skipped", func(t *testing.T) {
		r := resolve.Chain(
			resolve.Vars(map[string]string{"PORT": "3000"}),
			nil,
			resolve.Vars(map[string]string{"HOST": "localhost"}),
		)
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "3000", val)
	})

	t.Run("three-layer priority: map, env file, os prefix", func(t *testing.T) {
		t.Setenv("APP_PORT", "9999")
		r := resolve.Chain(
			resolve.Vars(map[string]string{"PORT": "3000", "HOST": "default"}),
			resolve.Vars(map[string]string{"PORT": "5000"}),
			resolve.OSPrefix("APP_"),
		)

		// APP_PORT is set, OSPrefix wins
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "9999", val)

		// HOST is only in the first Vars
		val, ok = r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "default", val)

		// MISSING is not in any resolver
		_, ok = r("MISSING")
		assert.False(t, ok)
	})
}
