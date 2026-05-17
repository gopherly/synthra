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

package synthratest

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
	"gopherly.dev/synthra/source"
)

// Format identifies a config file extension for [WriteFile] and [LoadFile].
type Format string

const (
	YAML Format = "yaml"
	JSON Format = "json"
	TOML Format = "toml"
)

// Config constructs a [*synthra.Synthra] without calling Load.
// It fails the test if [synthra.New] returns an error.
func Config(t *testing.T, opts ...synthra.Option) *synthra.Synthra {
	t.Helper()
	cfg, err := synthra.New(opts...)
	require.NoError(t, err)
	return cfg
}

// Load constructs and loads a [*synthra.Synthra] using m as its primary source.
// It prepends [synthra.WithSource] with [source.NewMap](m) before opts.
// It uses [testing.T.Context]. For another context, use [Config] then
// [*synthra.Synthra.Load].
func Load(t *testing.T, m map[string]any, opts ...synthra.Option) *synthra.Synthra {
	t.Helper()
	all := append([]synthra.Option{synthra.WithSource(source.NewMap(m))}, opts...)
	cfg := Config(t, all...)
	require.NoError(t, cfg.Load(t.Context()))
	return cfg
}

// WriteFile writes content to config.<format> under [testing.T.TempDir]
// and returns the path.
// Cleanup is handled by [testing.T.TempDir].
func WriteFile(t *testing.T, format Format, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config."+string(format))
	require.NoError(t, os.WriteFile(path, content, 0o600))
	return path
}

// LoadFile builds and loads a Synthra from a temp file from [WriteFile].
func LoadFile(t *testing.T, format Format, content []byte) *synthra.Synthra {
	t.Helper()
	cfg := Config(t, synthra.WithFile(WriteFile(t, format, content)))
	require.NoError(t, cfg.Load(t.Context()))
	return cfg
}

// ErrSource returns a [synthra.Source] whose Load always fails with err.
// It panics if err is nil, because that would make Load return (nil, nil).
func ErrSource(err error) synthra.Source {
	if err == nil {
		panic("synthratest: ErrSource requires non-nil error")
	}
	return errSource{err: err}
}

type errSource struct{ err error }

func (s errSource) Load(context.Context) (map[string]any, error) {
	return nil, s.err
}

// Dumper is a recording [synthra.Dumper]. It is safe for concurrent use.
// The zero value is usable. Set [Dumper.Err] so [Dumper.Dump] returns that error.
type Dumper struct {
	Err error

	mu    sync.Mutex
	calls int
	last  map[string]any
}

// Dump records the call and optionally returns [Dumper.Err].
func (d *Dumper) Dump(_ context.Context, values *map[string]any) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	if values != nil && *values != nil {
		cp := make(map[string]any, len(*values))
		maps.Copy(cp, *values)
		d.last = cp
	}
	return d.Err
}

// Calls returns how many times Dump has been invoked.
func (d *Dumper) Calls() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

// Last returns a shallow copy of the last Dump values, or nil if none.
func (d *Dumper) Last() map[string]any {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.last
}

// FuncCodec implements [codec.Decoder] and [codec.Encoder] with function fields.
// Either field may be nil: nil [FuncCodec.DecodeFunc] is a no-op decode;
// nil [FuncCodec.EncodeFunc] returns an empty byte slice and a nil error.
type FuncCodec struct {
	DecodeFunc func(data []byte, v any) error
	EncodeFunc func(v any) ([]byte, error)
}

// Decode implements [codec.Decoder].
func (c *FuncCodec) Decode(data []byte, v any) error {
	if c.DecodeFunc == nil {
		return nil
	}
	return c.DecodeFunc(data, v)
}

// Encode implements [codec.Encoder].
func (c *FuncCodec) Encode(v any) ([]byte, error) {
	if c.EncodeFunc == nil {
		return []byte{}, nil
	}
	return c.EncodeFunc(v)
}

// AssertString asserts cfg.String(key) equals want.
func AssertString(t *testing.T, cfg *synthra.Synthra, key, want string) {
	t.Helper()
	got, err := cfg.String(key)
	require.NoError(t, err, "key %q", key)
	require.Equal(t, want, got, "key %q", key)
}

// AssertInt asserts cfg.Int(key) equals want.
func AssertInt(t *testing.T, cfg *synthra.Synthra, key string, want int) {
	t.Helper()
	got, err := cfg.Int(key)
	require.NoError(t, err, "key %q", key)
	require.Equal(t, want, got, "key %q", key)
}

// AssertBool asserts cfg.Bool(key) equals want.
func AssertBool(t *testing.T, cfg *synthra.Synthra, key string, want bool) {
	t.Helper()
	got, err := cfg.Bool(key)
	require.NoError(t, err, "key %q", key)
	require.Equal(t, want, got, "key %q", key)
}

// AssertStringSlice asserts cfg.StringSlice(key) equals want.
func AssertStringSlice(t *testing.T, cfg *synthra.Synthra, key string, want []string) {
	t.Helper()
	got, err := cfg.StringSlice(key)
	require.NoError(t, err, "key %q", key)
	require.Equal(t, want, got, "key %q", key)
}

// AssertDumped asserts [Dumper.Calls] is 1 and [Dumper.Last] equals want
// (shallow map equality).
func AssertDumped(t *testing.T, d *Dumper, want map[string]any) {
	t.Helper()
	require.Equal(t, 1, d.Calls(), "expected exactly one Dump call")
	require.Equal(t, want, d.Last())
}
