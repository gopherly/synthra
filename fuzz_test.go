// Copyright 2026 The Gopherly Authors
// Copyright 2025 Company.info B.V.
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
	"testing"

	"gopherly.dev/synthra/codec"
	"gopherly.dev/synthra/source"
)

// FuzzContentSourceJSON fuzzes JSON content parsing.
func FuzzContentSourceJSON(f *testing.F) {
	// Seed corpus with valid JSON inputs
	f.Add([]byte(`{"foo": "bar"}`))
	f.Add([]byte(`{"nested": {"key": "value"}}`))
	f.Add([]byte(`{"array": [1, 2, 3]}`))
	f.Add([]byte(`{"bool": true, "number": 42}`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, input []byte) {
		cfg, err := New(WithContent(input, codec.JSON))
		if err != nil {
			return
		}

		// Should not panic even with invalid input
		err = cfg.Load(context.Background())
		// Invalid JSON should return an error, not panic
		if err != nil {
			var configErr *ConfigError
			if !errors.As(err, &configErr) {
				// Error should be wrapped in ConfigError
				t.Logf("expected ConfigError, got %T: %v", err, err)
			}
		}
	})
}

// FuzzContentSourceYAML fuzzes YAML content parsing.
func FuzzContentSourceYAML(f *testing.F) {
	// Seed corpus with valid YAML inputs
	f.Add([]byte("foo: bar"))
	f.Add([]byte("nested:\n  key: value"))
	f.Add([]byte("array:\n  - 1\n  - 2\n  - 3"))
	f.Add([]byte("bool: true\nnumber: 42"))
	f.Add([]byte("{}"))

	f.Fuzz(func(t *testing.T, input []byte) {
		cfg, err := New(WithContent(input, codec.YAML))
		if err != nil {
			return
		}

		// Should not panic even with invalid input
		err = cfg.Load(context.Background())
		if err != nil {
			var configErr *ConfigError
			if !errors.As(err, &configErr) {
				t.Logf("expected ConfigError, got %T: %v", err, err)
			}
		}
	})
}

// FuzzContentSourceTOML fuzzes TOML content parsing.
func FuzzContentSourceTOML(f *testing.F) {
	// Seed corpus with valid TOML inputs
	f.Add([]byte(`foo = "bar"`))
	f.Add([]byte("[nested]\nkey = \"value\""))
	f.Add([]byte("array = [1, 2, 3]"))
	f.Add([]byte("bool = true\nnumber = 42"))

	f.Fuzz(func(t *testing.T, input []byte) {
		cfg, err := New(WithContent(input, codec.TOML))
		if err != nil {
			return
		}

		// Should not panic even with invalid input
		err = cfg.Load(context.Background())
		if err != nil {
			var configErr *ConfigError
			if !errors.As(err, &configErr) {
				t.Logf("expected ConfigError, got %T: %v", err, err)
			}
		}
	})
}

// FuzzGet fuzzes key retrieval with dot notation.
func FuzzGet(f *testing.F) {
	// Seed corpus with various key patterns
	f.Add("foo")
	f.Add("foo.bar")
	f.Add("foo.bar.baz")
	f.Add("a.b.c.d.e")
	f.Add("")
	f.Add(".")
	f.Add("..")
	f.Add("foo.")
	f.Add(".foo")

	src := source.NewMap(map[string]any{
		"foo": "bar",
		"nested": map[string]any{
			"key": "value",
			"deep": map[string]any{
				"val": 42,
			},
		},
	})
	cfg, err := New(WithSource(src))
	if err != nil {
		f.Fatal(err)
	}
	if err = cfg.Load(context.Background()); err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, key string) {
		// Should not panic with any key input
		_ = cfg.Get(key)
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.String(key)
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Int(key)
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Bool(key)
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.StringSlice(key)
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.StringMap(key)
	})
}

// FuzzGetWithSpecialChars fuzzes key retrieval with special characters.
func FuzzGetWithSpecialChars(f *testing.F) {
	// Seed corpus with keys containing special characters
	f.Add("foo-bar")
	f.Add("foo_bar")
	f.Add("foo:bar")
	f.Add("foo/bar")
	f.Add("foo\\bar")
	f.Add("foo bar")
	f.Add("foo\tbar")
	f.Add("foo\nbar")

	src := source.NewMap(map[string]any{
		"foo-bar": "value1",
		"foo_bar": "value2",
	})
	cfg, err := New(WithSource(src))
	if err != nil {
		f.Fatal(err)
	}
	if err = cfg.Load(context.Background()); err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, key string) {
		// Should not panic with any key input
		_ = cfg.Get(key)
	})
}

// FuzzValidator fuzzes custom validator functions.
func FuzzValidator(f *testing.F) {
	// Seed corpus with various validation inputs
	f.Add("valid")
	f.Add("invalid")
	f.Add("")
	f.Add("test123")

	f.Fuzz(func(t *testing.T, value string) {
		src := source.NewMap(map[string]any{"key": value})
		validator := func(cfg map[string]any) error {
			// Simple validation that should not panic
			if v, ok := cfg["key"].(string); ok && v == "" {
				return errors.New("key cannot be empty")
			}
			return nil
		}

		cfg, err := New(WithSource(src), WithValidator(validator))
		if err != nil {
			return
		}

		// Should not panic even with invalid input
		err = cfg.Load(context.Background())
		if err != nil {
			t.Logf("expected validation error for input %q: %v", value, err)
		}
	})
}

// FuzzBinding fuzzes struct binding with various inputs.
func FuzzBinding(f *testing.F) {
	// Seed corpus with various string values
	f.Add("test", 42)
	f.Add("", 0)
	f.Add("a", -1)
	f.Add("very long string value", 999999)

	type TestStruct struct {
		Foo string `synthra:"foo"`
		Bar int    `synthra:"bar"`
	}

	f.Fuzz(func(t *testing.T, fooVal string, barVal int) {
		src := source.NewMap(map[string]any{"foo": fooVal, "bar": barVal})
		var bind TestStruct
		cfg, err := New(WithSource(src), WithBinding(&bind))
		if err != nil {
			return
		}

		// Should not panic with any input
		err = cfg.Load(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	})
}

// FuzzNormalizeMapKeys fuzzes the key normalization function.
func FuzzNormalizeMapKeys(f *testing.F) {
	// Seed corpus with various key patterns
	f.Add("FooBar")
	f.Add("foo_bar")
	f.Add("FOO-BAR")
	f.Add("CamelCase")
	f.Add("UPPERCASE")
	f.Add("lowercase")

	f.Fuzz(func(t *testing.T, key string) {
		// Create a map with the fuzzed key
		input := map[string]any{
			key: "value",
		}

		// Should not panic with any key input
		normalized := normalizeMapKeys(input)
		_ = normalized
	})
}

// FuzzGetTypedValues fuzzes type conversion functions.
func FuzzGetTypedValues(f *testing.F) {
	// Seed corpus with various value types
	f.Add("string", int64(42), float64(3.14), true)
	f.Add("", int64(0), float64(0), false)
	f.Add("test", int64(-1), float64(-1.5), true)

	f.Fuzz(func(t *testing.T, strVal string, intVal int64, floatVal float64, boolVal bool) {
		src := source.NewMap(map[string]any{
			"str":   strVal,
			"int":   intVal,
			"float": floatVal,
			"bool":  boolVal,
		})
		cfg, err := New(WithSource(src))
		if err != nil {
			return
		}
		if err = cfg.Load(context.Background()); err != nil {
			return
		}

		// Should not panic with any value types
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.String("str")
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Int("int")
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Int64("int")
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Float64("float")
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Bool("bool")

		// Try cross-type conversions (should not panic)
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.String("int")
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Int("str")
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Bool("str")
		//nolint:errcheck // fuzz: only panics are failures
		_, _ = cfg.Float64("str")
	})
}
