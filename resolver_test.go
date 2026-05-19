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

func TestResolverOr(t *testing.T) {
	t.Run("receiver wins when it finds the key", func(t *testing.T) {
		r := FromMap(map[string]string{"PORT": "3000"}).
			Or(FromMap(map[string]string{"PORT": "8080"}))
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "3000", val)
	})

	t.Run("falls through to fallback when receiver not found", func(t *testing.T) {
		r := FromMap(map[string]string{"HOST": "primary"}).
			Or(FromMap(map[string]string{"PORT": "8080"}))
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})

	t.Run("not found when no resolver has the key", func(t *testing.T) {
		r := FromMap(map[string]string{"PORT": "3000"}).
			Or(FromMap(map[string]string{"HOST": "localhost"}))
		val, ok := r("MISSING")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("Or with no fallbacks returns receiver unchanged", func(t *testing.T) {
		base := FromMap(map[string]string{"PORT": "3000"})
		r := base.Or()
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "3000", val)
	})

	t.Run("nil fallback in list is skipped", func(t *testing.T) {
		r := FromMap(map[string]string{"HOST": "primary"}).
			Or(nil, FromMap(map[string]string{"PORT": "8080"}))
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "8080", val)
	})

	t.Run("nil receiver is safe, falls through to fallback", func(t *testing.T) {
		r := Resolver(nil).Or(FromMap(map[string]string{"PORT": "9090"}))
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "9090", val)
	})

	t.Run("empty string is found and stops the chain", func(t *testing.T) {
		// A resolver that explicitly returns ("", true) must short-circuit;
		// the fallback must not be consulted.
		explicit := Resolver(func(name string) (string, bool) {
			if name == "VAR" {
				return "", true
			}
			return "", false
		})
		r := explicit.Or(FromMap(map[string]string{"VAR": "from-fallback"}))
		val, ok := r("VAR")
		assert.True(t, ok)
		assert.Equal(t, "", val) // fallback must not have overridden it
	})

	t.Run("three-layer priority (first wins): os-prefix > env-file > defaults", func(t *testing.T) {
		t.Setenv("APP_PORT", "9999")
		defaults := FromMap(map[string]string{"PORT": "3000", "HOST": "default"})
		middle := FromMap(map[string]string{"PORT": "5000"})
		r := FromEnv().Prefix("APP_").Or(middle).Or(defaults)

		// APP_PORT=9999 is set, Prefix("APP_") wins
		val, ok := r("PORT")
		assert.True(t, ok)
		assert.Equal(t, "9999", val)

		// HOST is only in defaults, not in middle or APP_*
		val, ok = r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "default", val)

		// MISSING is not in any resolver
		_, ok = r("MISSING")
		assert.False(t, ok)
	})

	t.Run("composition with Prefix", func(t *testing.T) {
		t.Setenv("APP_HOST", "os.example.com")
		envMap := FromMap(map[string]string{"APP_HOST": "map.example.com"})
		r := FromEnv().Prefix("APP_").Or(envMap.Prefix("APP_"))

		// OS env has APP_HOST → wins
		val, ok := r("HOST")
		assert.True(t, ok)
		assert.Equal(t, "os.example.com", val)
	})

	t.Run("second fallback reached when first fallback also misses", func(t *testing.T) {
		r := FromMap(map[string]string{}).
			Or(FromMap(map[string]string{})).
			Or(FromMap(map[string]string{"KEY": "found"}))
		val, ok := r("KEY")
		assert.True(t, ok)
		assert.Equal(t, "found", val)
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
