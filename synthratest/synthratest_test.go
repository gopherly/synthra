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

package synthratest

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
	"gopherly.dev/synthra/source"
)

func TestErrSource_loadFails(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("source load failed")
	src := ErrSource(loadErr)
	require.NotNil(t, src)

	cfg := Config(t, synthra.WithSource(src))
	err := cfg.Load(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "source load failed")
}

func TestErrSource_nilPanics(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() { ErrSource(nil) })
}

func TestDumper_withError(t *testing.T) {
	t.Parallel()

	dumpErr := errors.New("dumper write failed")
	dumper := &Dumper{Err: dumpErr}

	cfg := Config(t,
		synthra.WithSource(source.NewMap(map[string]any{"foo": "bar"})),
		synthra.WithDumper(dumper),
	)
	require.NoError(t, cfg.Load(context.Background()))

	err := cfg.Dump(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "dumper write failed")
}

func TestWriteFile_yaml(t *testing.T) {
	t.Parallel()

	content := []byte("key: value\nnested:\n  num: 42")
	path := WriteFile(t, YAML, content)
	require.NotEmpty(t, path)

	//nolint:gosec // G304: path is from WriteFile (t.TempDir() + fixed name), not user input
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestWriteFile_json(t *testing.T) {
	t.Parallel()

	content := []byte(`{"key":"value","nested":{"num":42}}`)
	path := WriteFile(t, JSON, content)
	require.NotEmpty(t, path)

	//nolint:gosec // G304: path is from WriteFile (t.TempDir() + fixed name), not user input
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestWriteFile_toml(t *testing.T) {
	t.Parallel()

	content := []byte("key = \"value\"\n[nested]\nnum = 42")
	path := WriteFile(t, TOML, content)
	require.NotEmpty(t, path)

	//nolint:gosec // G304: path is from WriteFile (t.TempDir() + fixed name), not user input
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestLoadFile_yaml(t *testing.T) {
	t.Parallel()

	content := []byte("app: myapp\nport: 8080")
	cfg := LoadFile(t, YAML, content)
	require.NotNil(t, cfg)
	s, err := cfg.String("app")
	require.NoError(t, err)
	assert.Equal(t, "myapp", s)
	p, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, p)
}

func TestLoadFile_json(t *testing.T) {
	t.Parallel()

	content := []byte(`{"app":"myapp","port":8080}`)
	cfg := LoadFile(t, JSON, content)
	require.NotNil(t, cfg)
	s, err := cfg.String("app")
	require.NoError(t, err)
	assert.Equal(t, "myapp", s)
	p, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, p)
}

func TestLoadFile_toml(t *testing.T) {
	t.Parallel()

	content := []byte("app = \"myapp\"\nport = 8080")
	cfg := LoadFile(t, TOML, content)
	require.NotNil(t, cfg)
	s, err := cfg.String("app")
	require.NoError(t, err)
	assert.Equal(t, "myapp", s)
	p, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, p)
}

func TestAssertString_int_bool_slice(t *testing.T) {
	t.Parallel()

	cfg := Load(t, map[string]any{
		"foo":  "bar",
		"num":  42,
		"on":   true,
		"tags": []string{"a", "b"},
	})
	AssertString(t, cfg, "foo", "bar")
	AssertInt(t, cfg, "num", 42)
	AssertBool(t, cfg, "on", true)
	AssertStringSlice(t, cfg, "tags", []string{"a", "b"})
}

func TestGet_inlineEqual(t *testing.T) {
	t.Parallel()

	cfg := Load(t, map[string]any{"foo": "bar", "num": 42})
	require.Equal(t, "bar", cfg.Get("foo"))
	require.Equal(t, 42, cfg.Get("num"))
}

func TestFuncCodec(t *testing.T) {
	t.Parallel()

	t.Run("Decode and Encode succeed", func(t *testing.T) {
		t.Parallel()

		decodeCalled := false
		encodeCalled := false
		mock := &FuncCodec{
			DecodeFunc: func(data []byte, v any) error {
				decodeCalled = true
				return nil
			},
			EncodeFunc: func(v any) ([]byte, error) {
				encodeCalled = true
				return []byte("encoded"), nil
			},
		}

		var dst map[string]any
		err := mock.Decode([]byte("input"), &dst)
		require.NoError(t, err)
		assert.True(t, decodeCalled)

		out, err := mock.Encode(map[string]any{"x": 1})
		require.NoError(t, err)
		assert.True(t, encodeCalled)
		assert.Equal(t, []byte("encoded"), out)
	})

	t.Run("Decode returns error", func(t *testing.T) {
		t.Parallel()

		decodeErr := errors.New("decode failed")
		mock := &FuncCodec{
			DecodeFunc: func([]byte, any) error { return decodeErr },
		}

		var dst map[string]any
		err := mock.Decode([]byte("x"), &dst)
		require.Error(t, err)
		assert.ErrorContains(t, err, "decode failed")
	})

	t.Run("Encode returns error", func(t *testing.T) {
		t.Parallel()

		encodeErr := errors.New("encode failed")
		mock := &FuncCodec{
			EncodeFunc: func(any) ([]byte, error) { return nil, encodeErr },
		}

		_, err := mock.Encode(map[string]any{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "encode failed")
	})

	t.Run("nil EncodeFunc returns empty bytes", func(t *testing.T) {
		t.Parallel()
		mock := &FuncCodec{DecodeFunc: func([]byte, any) error { return nil }}
		out, err := mock.Encode(nil)
		require.NoError(t, err)
		assert.Equal(t, []byte{}, out)
	})
}

func TestConfig_example(t *testing.T) {
	t.Parallel()
	cfg := Config(t,
		synthra.WithSource(source.NewMap(map[string]any{"k": "v"})),
	)
	require.NoError(t, cfg.Load(t.Context()))
	AssertString(t, cfg, "k", "v")
}
