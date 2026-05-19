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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromMap(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		r := FromMap(map[string]string{"PORT": "8080", "HOST": "localhost"})
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})

	t.Run("not found", func(t *testing.T) {
		r := FromMap(map[string]string{"PORT": "8080"})
		val, ok := r("MISSING")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("nil map", func(t *testing.T) {
		r := FromMap(nil)
		val, ok := r("PORT")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("empty map", func(t *testing.T) {
		r := FromMap(map[string]string{})
		val, ok := r("PORT")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("keys are case-sensitive", func(t *testing.T) {
		r := FromMap(map[string]string{"port": "8080"})
		_, ok := r("PORT")
		assert.False(t, ok)
		val, ok := r("port")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})
}

func TestFromEnv(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		t.Setenv("SYNTHRA_TEST_VAR", "hello")
		r := FromEnv()
		val, ok := r("SYNTHRA_TEST_VAR")
		assert.True(t, ok)
		assert.Equal(t, "hello", val)
	})

	t.Run("not found", func(t *testing.T) {
		r := FromEnv()
		val, ok := r("SYNTHRA_DEFINITELY_NOT_SET_XYZ123")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("reads live environment", func(t *testing.T) {
		r := FromEnv()
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

func TestResolverPrefix(t *testing.T) {
	t.Run("found with prefix stripped on FromEnv", func(t *testing.T) {
		t.Setenv("APP_PORT", "9090")
		r := FromEnv().Prefix("APP_")
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "9090", val)
	})

	t.Run("not found when prefixed var not set", func(t *testing.T) {
		r := FromEnv().Prefix("APP_")
		val, ok := r("SYNTHRA_DEFINITELY_NOT_SET_XYZ123")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("unprefixed var is not found", func(t *testing.T) {
		t.Setenv("PORT", "3000")
		r := FromEnv().Prefix("APP_")
		// Prefix("APP_") looks for APP_PORT, not PORT
		_, ok := r("PORT")
		assert.False(t, ok)
	})

	t.Run("empty prefix returns receiver unchanged", func(t *testing.T) {
		t.Setenv("SYNTHRA_EMPTY_PREFIX_VAR", "value")
		r := FromEnv().Prefix("")
		val, ok := r("SYNTHRA_EMPTY_PREFIX_VAR")
		assert.True(t, ok)
		assert.Equal(t, "value", val)
	})

	t.Run("reads live environment", func(t *testing.T) {
		t.Setenv("DPY_VAR_HOST", "db.local")
		r := FromEnv().Prefix("DPY_VAR_")
		val, ok := r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "db.local", val)
	})

	t.Run("prefix on FromMap", func(t *testing.T) {
		r := FromMap(map[string]string{"APP_PORT": "8080", "APP_HOST": "localhost"}).Prefix("APP_")
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)

		val, ok = r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "localhost", val)

		_, ok = r("MISSING")
		assert.False(t, ok)
	})

	t.Run("prefix on FromEnvFile", func(t *testing.T) {
		path := writeTempEnvFile(t, "SVC_PORT=9090\nSVC_HOST=svc.local\n")
		base, err := FromEnvFile(path)
		require.NoError(t, err)

		r := base.Prefix("SVC_")
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "9090", val)

		val, ok = r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "svc.local", val)
	})
}

func TestChainResolvers(t *testing.T) {
	t.Run("last resolver wins", func(t *testing.T) {
		r := chainResolvers(
			FromMap(map[string]string{"PORT": "3000"}),
			FromMap(map[string]string{"PORT": "8080"}),
		)
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})

	t.Run("falls back to earlier resolver", func(t *testing.T) {
		r := chainResolvers(
			FromMap(map[string]string{"PORT": "3000"}),
			FromMap(map[string]string{"HOST": "localhost"}),
		)
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "3000", val)
	})

	t.Run("not found when no resolver has the key", func(t *testing.T) {
		r := chainResolvers(
			FromMap(map[string]string{"PORT": "3000"}),
			FromMap(map[string]string{"HOST": "localhost"}),
		)
		val, ok := r("MISSING")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("no resolvers returns not found", func(t *testing.T) {
		r := chainResolvers()
		val, ok := r("PORT")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("nil resolver in chain is skipped", func(t *testing.T) {
		r := chainResolvers(
			FromMap(map[string]string{"PORT": "3000"}),
			nil,
			FromMap(map[string]string{"HOST": "localhost"}),
		)
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "3000", val)
	})

	t.Run("three-layer priority: map, map, os prefix", func(t *testing.T) {
		t.Setenv("APP_PORT", "9999")
		r := chainResolvers(
			FromMap(map[string]string{"PORT": "3000", "HOST": "default"}),
			FromMap(map[string]string{"PORT": "5000"}),
			FromEnv().Prefix("APP_"),
		)

		// APP_PORT is set, Prefix("APP_") wins
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "9999", val)

		// HOST is only in the first FromMap
		val, ok = r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "default", val)

		// MISSING is not in any resolver
		_, ok = r("MISSING")
		assert.False(t, ok)
	})
}

func TestFromEnvFile(t *testing.T) {
	t.Run("simple key=value", func(t *testing.T) {
		path := writeTempEnvFile(t, "FOO=bar\nBAZ=qux\n")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		val, ok := r("FOO")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)

		val, ok = r("BAZ")
		assert.True(t, ok)
		assert.Equal(t, "qux", val)
	})

	t.Run("double-quoted values", func(t *testing.T) {
		path := writeTempEnvFile(t, `FOO="hello world"`+"\n")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		val, ok := r("FOO")
		assert.True(t, ok)
		assert.Equal(t, "hello world", val)
	})

	t.Run("single-quoted values", func(t *testing.T) {
		path := writeTempEnvFile(t, "FOO='hello world'\n")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		val, ok := r("FOO")
		assert.True(t, ok)
		assert.Equal(t, "hello world", val)
	})

	t.Run("comment lines and blank lines are skipped", func(t *testing.T) {
		path := writeTempEnvFile(t, "# comment\n\nFOO=bar\n# another\nBAZ=qux\n")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		val, ok := r("FOO")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)

		val, ok = r("BAZ")
		assert.True(t, ok)
		assert.Equal(t, "qux", val)
	})

	t.Run("export prefix stripped", func(t *testing.T) {
		path := writeTempEnvFile(t, "export FOO=bar\nexport BAZ=qux\n")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		val, ok := r("FOO")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)
	})

	t.Run("inline comment stripped", func(t *testing.T) {
		path := writeTempEnvFile(t, "FOO=bar # this is a comment\n")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		val, ok := r("FOO")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)
	})

	t.Run("not found for missing key", func(t *testing.T) {
		path := writeTempEnvFile(t, "FOO=bar\n")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		_, ok := r("MISSING")
		assert.False(t, ok)
	})

	t.Run("file not found returns error", func(t *testing.T) {
		_, err := FromEnvFile("/nonexistent/path/.env")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "FromEnvFile")
	})

	t.Run("malformed line returns error", func(t *testing.T) {
		path := writeTempEnvFile(t, "NOEQUALSSIGN\n")
		_, err := FromEnvFile(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "FromEnvFile")
	})

	t.Run("empty file returns resolver that never finds", func(t *testing.T) {
		path := writeTempEnvFile(t, "")
		r, err := FromEnvFile(path)
		require.NoError(t, err)

		_, ok := r("ANY")
		assert.False(t, ok)
	})
}

// writeTempEnvFile writes content to a temp file and returns its path.
func writeTempEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
