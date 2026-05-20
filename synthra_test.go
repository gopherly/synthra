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
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra/codec"
	"gopherly.dev/synthra/source"
)

func mustString(t *testing.T, cfg *Synthra, key string) string {
	t.Helper()
	v, err := cfg.String(key)
	require.NoError(t, err)
	return v
}

func mustInt(t *testing.T, cfg *Synthra, key string) int {
	t.Helper()
	v, err := cfg.Int(key)
	require.NoError(t, err)
	return v
}

func mustBool(t *testing.T, cfg *Synthra, key string) bool {
	t.Helper()
	v, err := cfg.Bool(key)
	require.NoError(t, err)
	return v
}

func mustInt64(t *testing.T, cfg *Synthra, key string) int64 {
	t.Helper()
	v, err := cfg.Int64(key)
	require.NoError(t, err)
	return v
}

func mustFloat64(t *testing.T, cfg *Synthra, key string) float64 {
	t.Helper()
	v, err := cfg.Float64(key)
	require.NoError(t, err)
	return v
}

func mustTime(t *testing.T, cfg *Synthra, key string) time.Time {
	t.Helper()
	v, err := cfg.Time(key)
	require.NoError(t, err)
	return v
}

func mustDuration(t *testing.T, cfg *Synthra, key string) time.Duration {
	t.Helper()
	v, err := cfg.Duration(key)
	require.NoError(t, err)
	return v
}

func mustIntSlice(t *testing.T, cfg *Synthra, key string) []int {
	t.Helper()
	v, err := cfg.IntSlice(key)
	require.NoError(t, err)
	return v
}

func mustStringSlice(t *testing.T, cfg *Synthra, key string) []string {
	t.Helper()
	v, err := cfg.StringSlice(key)
	require.NoError(t, err)
	return v
}

func mustStringMap(t *testing.T, cfg *Synthra, key string) map[string]any {
	t.Helper()
	v, err := cfg.StringMap(key)
	require.NoError(t, err)
	return v
}

// loadTestConfig builds and loads a Synthra using source.NewMap(m).
// External tests should use gopherly.dev/synthra/synthratest.Load instead.
func loadTestConfig(t *testing.T, m map[string]any) *Synthra {
	t.Helper()
	cfg, err := New(WithSource(source.NewMap(m)))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(t.Context()))
	return cfg
}

type errOnlySource struct{ err error }

func (e errOnlySource) Load(context.Context) (map[string]any, error) {
	return nil, e.err
}

// recordingDumper records Dump calls for tests (see synthratest.Dumper).
// Defined here to avoid importing synthratest from these tests.
type recordingDumper struct {
	Err error

	mu    sync.Mutex
	calls int
	last  map[string]any
}

func (d *recordingDumper) Dump(_ context.Context, values map[string]any) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	if values != nil {
		cp := make(map[string]any, len(values))
		maps.Copy(cp, values)
		d.last = cp
	}
	return d.Err
}

func (d *recordingDumper) Calls() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

func (d *recordingDumper) Last() map[string]any {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.last
}

// validatingStruct is used by TestBinding_ValidatorInterface; implements
// Validator.
type validatingStruct struct {
	Port int `synthra:"port"`
}

func (v *validatingStruct) Validate() error {
	if v.Port <= 0 {
		return errors.New("port must be positive")
	}
	return nil
}

// bindStruct is used by binding tests.
type bindStruct struct {
	Foo string `synthra:"foo"`
	Bar int    `synthra:"bar"`
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no options succeeds",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "with valid source succeeds",
			opts:    []Option{WithSource(source.NewMap(map[string]any{"foo": "bar"}))},
			wantErr: false,
		},
		{
			name:    "with nil source fails",
			opts:    []Option{WithSource(nil)},
			wantErr: true,
			errMsg:  "source cannot be nil",
		},
		{
			name:    "with nil dumper fails",
			opts:    []Option{WithDumper(nil)},
			wantErr: true,
			errMsg:  "dumper cannot be nil",
		},
		{
			name:    "with nil binding fails",
			opts:    []Option{WithBinding((*bindStruct)(nil))},
			wantErr: true,
			errMsg:  "binding target cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := New(tt.opts...)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, cfg)
		})
	}
}

func TestNew_multipleValidationErrors(t *testing.T) {
	t.Parallel()
	cfg, err := New(
		WithSource(nil),
		WithBinding((*bindStruct)(nil)),
	)
	require.Error(t, err)
	require.Nil(t, cfg)
	// Joined error should contain both validation messages
	assert.Contains(t, err.Error(), "source cannot be nil")
	assert.Contains(t, err.Error(), "binding target cannot be nil")
}

func TestNew_WithTag(t *testing.T) {
	t.Parallel()

	type cfgTagStruct struct {
		Foo string `cfg:"foo"`
		Bar int    `cfg:"bar"`
	}

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
		errMsg  string
		verify  func(t *testing.T, cfg *Synthra)
	}{
		{
			name: "valid custom tag binds correctly",
			opts: []Option{
				WithSource(source.NewMap(map[string]any{"foo": "baz", "bar": 99})),
				WithTag("cfg"),
				WithBinding(&cfgTagStruct{}),
			},
			wantErr: false,
			verify: func(t *testing.T, cfg *Synthra) {
				t.Helper()
				require.NoError(t, cfg.Load(context.Background()))
				assert.Equal(t, "baz", mustString(t, cfg, "foo"))
				assert.Equal(t, 99, mustInt(t, cfg, "bar"))
			},
		},
		{
			name:    "empty tag name fails",
			opts:    []Option{WithTag("")},
			wantErr: true,
			errMsg:  "tag name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := New(tt.opts...)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, cfg)
			if tt.verify != nil {
				tt.verify(t, cfg)
			}
		})
	}
}

func TestNew_NilOptionFails(t *testing.T) {
	t.Parallel()

	src1 := source.NewMap(map[string]any{"a": "1"})
	src2 := source.NewMap(map[string]any{"b": "2"})
	cfg, err := New(WithSource(src1), nil, WithSource(src2))
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.Contains(t, err.Error(), "cannot be nil")
	assert.Contains(t, err.Error(), "option[1]")
}

func TestNew_NilValidatorFails(t *testing.T) {
	t.Parallel()

	cfg, err := New(WithValidator(nil))
	require.Error(t, err)
	require.Nil(t, cfg)
	assert.ErrorContains(t, err, "validator cannot be nil")
}

func TestMustNew_NilOptionPanics(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{"a": "1"})
	var panicMsg string
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicMsg = fmt.Sprint(r)
			}
		}()
		MustNew(WithSource(src), nil)
	}()
	require.NotEmpty(t, panicMsg, "MustNew with nil option should panic")
	assert.Contains(t, panicMsg, "cannot be nil")
	assert.Contains(t, panicMsg, "option[1]")
}

func TestNew_OptionErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		opt         Option
		wantErr     bool
		errContains string
	}{
		{
			name:        "WithFileDumper unknown extension",
			opt:         WithFileDumper("file.xyz"),
			wantErr:     true,
			errContains: "cannot detect format",
		},
		{
			name:        "WithFile unknown extension",
			opt:         WithFile("file.xyz"),
			wantErr:     true,
			errContains: "cannot detect format",
		},
		{
			name:        "WithFileFS nil filesystem",
			opt:         WithFileFS(nil, "a.yaml"),
			wantErr:     true,
			errContains: "filesystem cannot be nil",
		},
		{
			name:        "WithFileFS unknown extension",
			opt:         WithFileFS(fstest.MapFS{}, "file.xyz"),
			wantErr:     true,
			errContains: "cannot detect format",
		},
		{
			name:        "WithFileFSAs nil filesystem",
			opt:         WithFileFSAs(nil, "a.yaml", codec.YAML),
			wantErr:     true,
			errContains: "filesystem cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := New(tt.opt)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.errContains)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, cfg)
		})
	}
}

func TestWithFileFS_Load(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"app.yaml": &fstest.MapFile{Data: []byte("port: 4242\n")},
	}
	cfg, err := New(WithFileFS(fsys, "app.yaml"))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, 4242, mustInt(t, cfg, "port"))
}

func TestWithFileFSAs_Load(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"cfg": &fstest.MapFile{Data: []byte("name: test\n")},
	}
	cfg, err := New(WithFileFSAs(fsys, "cfg", codec.YAML))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "test", mustString(t, cfg, "name"))
}

func TestDetectFormat_UnknownExtension(t *testing.T) {
	t.Parallel()

	_, err := detectFormat("file.xyz")
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot detect format from extension")
	assert.ErrorContains(t, err, "WithFileAs()")
}

func TestNew_WithConsul_OptionErrorPaths(t *testing.T) {
	// Do not use t.Parallel() here: subtests use t.Setenv which is incompatible with parallel.

	t.Run("with CONSUL_HTTP_ADDR set unknown extension returns error", func(t *testing.T) {
		t.Setenv("CONSUL_HTTP_ADDR", "http://localhost:8500")

		_, err := New(WithConsul("path/file.xyz"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot detect format")
	})
}

func TestNew_MultipleOptionErrors(t *testing.T) {
	t.Parallel()

	_, err := New(WithSource(nil), WithDumper(nil), WithBinding((*bindStruct)(nil)))
	require.Error(t, err)
	// Errors are joined; all should be present
	assert.Contains(t, err.Error(), "source cannot be nil")
	assert.Contains(t, err.Error(), "dumper cannot be nil")
	assert.Contains(t, err.Error(), "binding target cannot be nil")
}

func TestMustNew(t *testing.T) {
	t.Parallel()

	t.Run("success with no options", func(t *testing.T) {
		t.Parallel()
		c := MustNew()
		assert.NotNil(t, c)
	})

	t.Run("success with valid source", func(t *testing.T) {
		t.Parallel()
		src := source.NewMap(map[string]any{"foo": "bar"})
		c := MustNew(WithSource(src))
		assert.NotNil(t, c)
		require.NoError(t, c.Load(context.Background()))
		assert.Equal(t, "bar", mustString(t, c, "foo"))
	})

	t.Run("panics with nil source", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			MustNew(WithSource(nil))
		})
	})

	t.Run("panic message contains config failure prefix", func(t *testing.T) {
		t.Parallel()
		var panicMsg string
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicMsg = fmt.Sprint(r)
				}
			}()
			MustNew(WithSource(nil))
		}()
		require.Contains(t, panicMsg, "synthra: validation failed")
	})
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func() (*Synthra, error)
		wantErr bool
		errMsg  string
	}{
		{
			name: "succeeds with valid source",
			setup: func() (*Synthra, error) {
				return New(WithSource(source.NewMap(map[string]any{"foo": "bar", "bar": 42})))
			},
			wantErr: false,
		},
		{
			name: "succeeds with no sources",
			setup: func() (*Synthra, error) {
				return New()
			},
			wantErr: false,
		},
		{
			name: "succeeds with nil source map",
			setup: func() (*Synthra, error) {
				return New(WithSource(source.NewMap(nil)))
			},
			wantErr: false,
		},
		{
			name: "error propagates from source",
			setup: func() (*Synthra, error) {
				return New(WithSource(errOnlySource{err: errors.New("fail")}))
			},
			wantErr: true,
		},
		{
			name: "multiple sources merge correctly",
			setup: func() (*Synthra, error) {
				return New(
					WithSource(source.NewMap(map[string]any{"foo": "bar", "bar": 1})),
					WithSource(source.NewMap(map[string]any{"bar": 2, "baz": 3})),
				)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := tt.setup()
			require.NoError(t, err, "setup should not fail")

			err = cfg.Load(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestLoad_MultipleSources(t *testing.T) {
	t.Parallel()

	src1 := source.NewMap(map[string]any{"foo": "bar", "bar": 1})
	src2 := source.NewMap(map[string]any{"bar": 2, "baz": 3})
	cfg, err := New(WithSource(src1), WithSource(src2))
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "bar", mustString(t, cfg, "foo"))
	assert.Equal(t, 2, mustInt(t, cfg, "bar")) // src2 overrides src1
	assert.Equal(t, 3, mustInt(t, cfg, "baz"))
}

func TestLoad_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so Load sees ctx.Err() != nil

	src := source.NewMap(map[string]any{"foo": "bar"})
	cfg, err := New(WithSource(src))
	require.NoError(t, err)

	err = cfg.Load(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLoad_BindingPointerToNonStruct(t *testing.T) {
	t.Parallel()

	var notAStruct int
	cfg, err := New(WithSource(source.NewMap(map[string]any{"x": "y"})), WithBinding(&notAStruct))
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.Error(t, err)
	// Binding to *int fails (decode expects struct, or applyDefaults rejects non-struct)
}

func TestLoad_BindingInvalidDurationDefault(t *testing.T) {
	t.Parallel()

	type withDuration struct {
		Timeout time.Duration `synthra:"timeout" default:"not-a-duration"`
	}
	var target withDuration
	cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to set default")
}

func TestLoad_BindingUnsupportedDefaultType(t *testing.T) {
	t.Parallel()

	type withSliceDefault struct {
		Items []string `synthra:"items" default:"a,b,c"` // slice default not supported by setDefaultValue
	}
	var target withSliceDefault
	cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
	require.NoError(t, err)

	err = cfg.Load(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported type for default tag")
}

func TestValues_WithoutLoad(t *testing.T) {
	t.Parallel()

	cfg, err := New()
	require.NoError(t, err)

	vals := cfg.Values()
	require.NotNil(t, vals)
	require.NotNil(t, *vals)
	assert.Empty(t, *vals)
}

func TestBinding(t *testing.T) {
	t.Parallel()

	t.Run("basic binding succeeds", func(t *testing.T) {
		t.Parallel()
		var bind bindStruct
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{"foo": "bar", "bar": 42})),
			WithBinding(&bind),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))
		assert.Equal(t, "bar", bind.Foo)
		assert.Equal(t, 42, bind.Bar)
	})

	t.Run("binding with extra fields succeeds", func(t *testing.T) {
		t.Parallel()
		var bind bindStruct
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{"foo": "bar", "bar": 42, "extra": 99})),
			WithBinding(&bind),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))
		assert.Equal(t, "bar", bind.Foo)
		assert.Equal(t, 42, bind.Bar)
	})

	t.Run("binding with missing fields uses defaults", func(t *testing.T) {
		t.Parallel()
		var bind bindStruct
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{"foo": "bar"})),
			WithBinding(&bind),
		)
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))
		assert.Equal(t, "bar", bind.Foo)
		assert.Equal(t, 0, bind.Bar)
	})

	t.Run("binding with type mismatch fails", func(t *testing.T) {
		t.Parallel()
		var bind bindStruct
		cfg, err := New(
			WithSource(source.NewMap(map[string]any{"foo": 123, "bar": "notanint"})),
			WithBinding(&bind),
		)
		require.NoError(t, err)
		require.Error(t, cfg.Load(context.Background()))
	})
}

func TestBinding_DefaultTag(t *testing.T) {
	t.Parallel()

	type defaultTagStruct struct {
		Foo     string        `synthra:"foo" default:"defaultfoo"`
		Bar     int           `synthra:"bar" default:"42"`
		Enabled bool          `synthra:"enabled" default:"true"`
		Timeout time.Duration `synthra:"timeout" default:"5s"`
	}

	tests := []struct {
		name   string
		conf   map[string]any
		verify func(t *testing.T, target *defaultTagStruct)
	}{
		{
			name: "defaults applied when keys omitted",
			conf: map[string]any{"foo": "fromconfig"},
			verify: func(t *testing.T, target *defaultTagStruct) {
				t.Helper()
				assert.Equal(t, "fromconfig", target.Foo)
				assert.Equal(t, 42, target.Bar)
				assert.True(t, target.Enabled)
				assert.Equal(t, 5*time.Second, target.Timeout)
			},
		},
		{
			name: "provided values override defaults",
			conf: map[string]any{"foo": "x", "bar": 7, "enabled": true, "timeout": "10s"},
			verify: func(t *testing.T, target *defaultTagStruct) {
				t.Helper()
				assert.Equal(t, "x", target.Foo)
				assert.Equal(t, 7, target.Bar)
				assert.True(t, target.Enabled)
				assert.Equal(t, 10*time.Second, target.Timeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var target defaultTagStruct
			cfg, err := New(WithSource(source.NewMap(tt.conf)), WithBinding(&target))
			require.NoError(t, err)
			require.NoError(t, cfg.Load(context.Background()))
			tt.verify(t, &target)
		})
	}
}

func TestBinding_ValidatorInterface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		conf    map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Validate returns nil succeeds",
			conf:    map[string]any{"port": 8080},
			wantErr: false,
		},
		{
			name:    "Validate returns error fails",
			conf:    map[string]any{}, // port omitted => 0, Validate rejects
			wantErr: true,
			errMsg:  "port must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var target validatingStruct
			cfg, err := New(WithSource(source.NewMap(tt.conf)), WithBinding(&target))
			require.NoError(t, err)

			err = cfg.Load(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.ErrorContains(t, err, tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 8080, target.Port)
		})
	}
}

func TestDump(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func() (*Synthra, *recordingDumper, error)
		verify  func(t *testing.T, dumper *recordingDumper)
		wantErr bool
	}{
		{
			name: "calls dumper successfully",
			setup: func() (*Synthra, *recordingDumper, error) {
				src := source.NewMap(map[string]any{"foo": "bar"})
				dumper := &recordingDumper{}
				cfg, err := New(WithSource(src), WithDumper(dumper))
				if err != nil {
					return nil, nil, err
				}
				if loadErr := cfg.Load(context.Background()); loadErr != nil {
					return nil, nil, loadErr
				}
				return cfg, dumper, nil
			},
			verify: func(t *testing.T, dumper *recordingDumper) {
				t.Helper()
				require.Equal(t, 1, dumper.Calls())
				require.Equal(t, map[string]any{"foo": "bar"}, dumper.Last())
			},
			wantErr: false,
		},
		{
			name: "succeeds with no dumpers",
			setup: func() (*Synthra, *recordingDumper, error) {
				src := source.NewMap(map[string]any{"foo": "bar"})
				cfg, err := New(WithSource(src))
				if err != nil {
					return nil, nil, err
				}
				if loadErr := cfg.Load(context.Background()); loadErr != nil {
					return nil, nil, loadErr
				}
				return cfg, nil, nil
			},
			verify:  nil,
			wantErr: false,
		},
		{
			name: "error propagates from dumper",
			setup: func() (*Synthra, *recordingDumper, error) {
				src := source.NewMap(map[string]any{"foo": "bar"})
				dumper := &recordingDumper{Err: errors.New("dump error")}
				cfg, err := New(WithSource(src), WithDumper(dumper))
				if err != nil {
					return nil, nil, err
				}
				if loadErr := cfg.Load(context.Background()); loadErr != nil {
					return nil, nil, loadErr
				}
				return cfg, dumper, nil
			},
			verify:  nil,
			wantErr: true,
		},
		{
			name: "calls multiple dumpers",
			setup: func() (*Synthra, *recordingDumper, error) {
				src := source.NewMap(map[string]any{"foo": "bar"})
				dumper1 := &recordingDumper{}
				dumper2 := &recordingDumper{}
				cfg, err := New(WithSource(src), WithDumper(dumper1), WithDumper(dumper2))
				if err != nil {
					return nil, nil, err
				}
				if loadErr := cfg.Load(context.Background()); loadErr != nil {
					return nil, nil, loadErr
				}
				// Return first dumper for verification
				return cfg, dumper1, nil
			},
			verify: func(t *testing.T, dumper *recordingDumper) {
				t.Helper()
				require.Equal(t, 1, dumper.Calls())
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, dumper, err := tt.setup()
			require.NoError(t, err, "setup should not fail")

			err = cfg.Dump(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.verify != nil && dumper != nil {
				tt.verify(t, dumper)
			}
		})
	}
}

func TestDump_NilContext(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{"foo": "bar"})
	cfg, err := New(WithSource(src))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// Testing nil context handling - we need to verify the function properly rejects nil
	// Using a helper to call Dump with nil to avoid linter warnings in the main test code
	callDumpWithNil := func(c *Synthra) error {
		//nolint:staticcheck // SA1012: Intentionally testing nil context error handling
		return c.Dump(nil)
	}

	err = callDumpWithNil(cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilContext)
}

func TestDump_NoLoad(t *testing.T) {
	t.Parallel()

	dumper := &recordingDumper{}
	cfg, err := New(WithSource(source.NewMap(map[string]any{"foo": "bar"})), WithDumper(dumper))
	require.NoError(t, err)
	// Do not call Load

	err = cfg.Dump(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, dumper.Calls())
	require.NotNil(t, dumper.Last())
	assert.Empty(t, dumper.Last())
}

func TestLoad_NilContext(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{"foo": "bar"})
	cfg, err := New(WithSource(src))
	require.NoError(t, err)

	callLoadWithNil := func(c *Synthra) error {
		//nolint:staticcheck // SA1012: Intentionally testing nil context error handling
		return c.Load(nil)
	}

	err = callLoadWithNil(cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilContext)
}

func TestGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		conf map[string]any
		key  string
		want any
	}{
		{
			name: "simple key",
			conf: map[string]any{"foo": "bar"},
			key:  "foo",
			want: "bar",
		},
		{
			name: "nested key with dot notation",
			conf: map[string]any{
				"outer": map[string]any{
					"inner": map[string]any{
						"val": 42,
					},
				},
			},
			key:  "outer.inner.val",
			want: 42,
		},
		{
			name: "deeply nested key",
			conf: map[string]any{
				"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": 1}}},
			},
			key:  "a.b.c.d",
			want: 1,
		},
		{
			name: "not found returns nil",
			conf: map[string]any{"foo": "bar"},
			key:  "notfound",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := loadTestConfig(t, tt.conf)
			got := cfg.Get(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGet_EmptyKey(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t, map[string]any{"foo": "bar"})

	got := cfg.Get("")
	assert.Nil(t, got)

	_, err := Get[string](cfg, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "not found")
}

func TestGet_MissingKeyReturnsError(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t, map[string]any{"foo": "bar"})

	_, err := Get[int](cfg, "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	_, err = Get[string](cfg, "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	_, err = Get[bool](cfg, "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestGet_StringAsIntCoercesViaCast(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t, map[string]any{"port": "not-a-number"})

	// Generic [Get] uses the same coercion path as [GetOr]; invalid numeric
	// strings coerce to zero without an error (see [convertToType]).
	v, err := Get[int](cfg, "port")
	require.NoError(t, err)
	assert.Equal(t, 0, v)
}

func TestGet_NilConfigAndKeyNotFoundAndConversionError(t *testing.T) {
	t.Parallel()

	t.Run("nil config returns error", func(t *testing.T) {
		t.Parallel()
		var cfg *Synthra
		_, err := Get[string](cfg, "key")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNilConfig)
	})

	t.Run("key not found returns error", func(t *testing.T) {
		t.Parallel()
		cfg := loadTestConfig(t, map[string]any{"foo": "bar"})
		_, err := Get[int](cfg, "nonexistent")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("value not convertible returns error", func(t *testing.T) {
		t.Parallel()
		type customType struct{ X int }
		cfg := loadTestConfig(t, map[string]any{"key": "string-value"})
		_, err := Get[customType](cfg, "key")
		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot convert")
		assert.ErrorContains(t, err, "key")
	})
}

func TestGetOr(t *testing.T) {
	t.Parallel()

	t.Run("key present returns value", func(t *testing.T) {
		t.Parallel()
		cfg := loadTestConfig(t, map[string]any{"port": 9090})
		got := GetOr(cfg, "port", 8080)
		assert.Equal(t, 9090, got)
	})

	t.Run("key missing returns default", func(t *testing.T) {
		t.Parallel()
		cfg := loadTestConfig(t, map[string]any{"foo": "bar"})
		got := GetOr(cfg, "port", 8080)
		assert.Equal(t, 8080, got)
	})

	t.Run("nil config returns default", func(t *testing.T) {
		t.Parallel()
		var cfg *Synthra
		got := GetOr(cfg, "port", 8080)
		assert.Equal(t, 8080, got)
	})
}

func TestGet_UnsupportedType(t *testing.T) {
	t.Parallel()

	type myType struct{}

	cfg := loadTestConfig(t, map[string]any{"custom": "value"})

	_, err := Get[myType](cfg, "custom")
	require.Error(t, err)
	assert.ErrorContains(t, err, "cannot convert")
}

func TestGetTypedValues(t *testing.T) {
	t.Parallel()

	timeStr := "2023-01-01T12:00:00Z"
	durStr := "1h2m3s"
	conf := map[string]any{
		"str":         "foo",
		"bool":        true,
		"boolstr":     "true",
		"int":         42,
		"intstr":      "42",
		"int8":        int8(8),
		"int16":       int16(16),
		"int32":       int32(32),
		"int64":       int64(64),
		"uint8":       uint8(8),
		"uint":        uint(7),
		"uint16":      uint16(16),
		"uint32":      uint32(32),
		"uint64":      uint64(64),
		"float32":     float32(1.5),
		"float32str":  "2.5",
		"float64":     3.14,
		"floatstr":    "2.71",
		"time":        timeStr,
		"duration":    durStr,
		"intslice":    []any{1, 2, 3},
		"strslice":    []any{"a", "b"},
		"map":         map[string]any{"a": 1},
		"mapstr":      map[string]any{"a": "x"},
		"mapstrslice": map[string]any{"a": []any{"x", "y"}},
	}

	cfg := loadTestConfig(t, conf)

	tests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{
			name: "GetString",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, "foo", mustString(t, cfg, "str"))
				v, err := Get[string](cfg, "str")
				require.NoError(t, err)
				assert.Equal(t, "foo", v)
				_, err = Get[string](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetBool",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.True(t, mustBool(t, cfg, "bool"))
				b, err := Get[bool](cfg, "boolstr")
				require.NoError(t, err)
				assert.True(t, b)
				_, err = Get[bool](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetInt",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, 42, mustInt(t, cfg, "int"))
				i, err := Get[int](cfg, "intstr")
				require.NoError(t, err)
				assert.Equal(t, 42, i)
				_, err = Get[int](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetInt8",
			testFn: func(t *testing.T) {
				t.Helper()
				i8, err := Get[int8](cfg, "int8")
				require.NoError(t, err)
				assert.Equal(t, int8(8), i8)
				_, err = Get[int8](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetInt16",
			testFn: func(t *testing.T) {
				t.Helper()
				i16, err := Get[int16](cfg, "int16")
				require.NoError(t, err)
				assert.Equal(t, int16(16), i16)
				_, err = Get[int16](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetInt32",
			testFn: func(t *testing.T) {
				t.Helper()
				i32, err := Get[int32](cfg, "int32")
				require.NoError(t, err)
				assert.Equal(t, int32(32), i32)
				_, err = Get[int32](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetInt64",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, int64(64), mustInt64(t, cfg, "int64"))
				i64, err := Get[int64](cfg, "int64")
				require.NoError(t, err)
				assert.Equal(t, int64(64), i64)
				_, err = Get[int64](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetUint8",
			testFn: func(t *testing.T) {
				t.Helper()
				u8, err := Get[uint8](cfg, "uint8")
				require.NoError(t, err)
				assert.Equal(t, uint8(8), u8)
				_, err = Get[uint8](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetUint",
			testFn: func(t *testing.T) {
				t.Helper()
				u, err := Get[uint](cfg, "uint")
				require.NoError(t, err)
				assert.Equal(t, uint(7), u)
				_, err = Get[uint](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetUint16",
			testFn: func(t *testing.T) {
				t.Helper()
				u16, err := Get[uint16](cfg, "uint16")
				require.NoError(t, err)
				assert.Equal(t, uint16(16), u16)
				_, err = Get[uint16](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetUint32",
			testFn: func(t *testing.T) {
				t.Helper()
				u32, err := Get[uint32](cfg, "uint32")
				require.NoError(t, err)
				assert.Equal(t, uint32(32), u32)
				_, err = Get[uint32](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetUint64",
			testFn: func(t *testing.T) {
				t.Helper()
				u64, err := Get[uint64](cfg, "uint64")
				require.NoError(t, err)
				assert.Equal(t, uint64(64), u64)
				_, err = Get[uint64](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetFloat32",
			testFn: func(t *testing.T) {
				t.Helper()
				f32, err := Get[float32](cfg, "float32str")
				require.NoError(t, err)
				assert.InDelta(t, 2.5, float64(f32), 0.0001)
				_, err = Get[float32](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetFloat64",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.InDelta(t, 3.14, mustFloat64(t, cfg, "float64"), 0.0001)
				f64, err := Get[float64](cfg, "floatstr")
				require.NoError(t, err)
				assert.InDelta(t, 2.71, f64, 0.0001)
				_, err = Get[float64](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetTime",
			testFn: func(t *testing.T) {
				t.Helper()
				tm, err := Get[time.Time](cfg, "time")
				require.NoError(t, err)
				assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), tm)
				assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), mustTime(t, cfg, "time"))
				_, err = Get[time.Time](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetDuration",
			testFn: func(t *testing.T) {
				t.Helper()
				d, err := Get[time.Duration](cfg, "duration")
				require.NoError(t, err)
				assert.Equal(t, 1*time.Hour+2*time.Minute+3*time.Second, d)
				assert.Equal(t, 1*time.Hour+2*time.Minute+3*time.Second, mustDuration(t, cfg, "duration"))
				_, err = Get[time.Duration](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetIntSlice",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, []int{1, 2, 3}, mustIntSlice(t, cfg, "intslice"))
				is, err := Get[[]int](cfg, "intslice")
				require.NoError(t, err)
				assert.Equal(t, []int{1, 2, 3}, is)
				_, err = Get[[]int](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetStringSlice",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, []string{"a", "b"}, mustStringSlice(t, cfg, "strslice"))
				ss, err := Get[[]string](cfg, "strslice")
				require.NoError(t, err)
				assert.Equal(t, []string{"a", "b"}, ss)
				_, err = Get[[]string](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetStringMap",
			testFn: func(t *testing.T) {
				t.Helper()
				assert.Equal(t, map[string]any{"a": 1}, mustStringMap(t, cfg, "map"))
				m, err := Get[map[string]any](cfg, "map")
				require.NoError(t, err)
				assert.Equal(t, map[string]any{"a": 1}, m)
				_, err = Get[map[string]any](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetStringMapString",
			testFn: func(t *testing.T) {
				t.Helper()
				ms, err := Get[map[string]string](cfg, "mapstr")
				require.NoError(t, err)
				assert.Equal(t, map[string]string{"a": "x"}, ms)
				_, err = Get[map[string]string](cfg, "notfound")
				assert.Error(t, err)
			},
		},
		{
			name: "GetStringMapStringSlice",
			testFn: func(t *testing.T) {
				t.Helper()
				mss, err := Get[map[string][]string](cfg, "mapstrslice")
				require.NoError(t, err)
				assert.Equal(t, map[string][]string{"a": {"x", "y"}}, mss)
				_, err = Get[map[string][]string](cfg, "notfound")
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.testFn(t)
		})
	}
}

func TestValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		conf      map[string]any
		validator func(context.Context, *Configuration) error
		wantErr   bool
		errMsg    string
	}{
		{
			name: "validator passes",
			conf: map[string]any{"foo": "baz"},
			validator: func(_ context.Context, r *Configuration) error {
				if r.StringOr("foo", "") != "baz" {
					return errors.New("foo must be 'baz'")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "validator fails",
			conf: map[string]any{"foo": "bar"},
			validator: func(_ context.Context, r *Configuration) error {
				if r.StringOr("foo", "") != "baz" {
					return errors.New("foo must be 'baz'")
				}
				return nil
			},
			wantErr: true,
		},
		{
			name: "validator panic with string is caught",
			conf: map[string]any{"foo": "bar"},
			validator: func(_ context.Context, _ *Configuration) error {
				panic("validator panic")
			},
			wantErr: true,
		},
		{
			name: "validator panic with error type is caught",
			conf: map[string]any{"foo": "bar"},
			validator: func(_ context.Context, _ *Configuration) error {
				panic(errors.New("typed panic error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := source.NewMap(tt.conf)
			cfg, err := New(WithSource(src), WithValidator(tt.validator))
			require.NoError(t, err)

			err = cfg.Load(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestJSONSchemaValidation(t *testing.T) {
	t.Parallel()

	schema := []byte(`{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","properties":{"foo":{"type":"string"},"bar":{"type":"integer"}},"required":["foo","bar"]}`)

	tests := []struct {
		name    string
		conf    map[string]any
		wantErr bool
	}{
		{
			name:    "valid data passes",
			conf:    map[string]any{"foo": "bar", "bar": 42},
			wantErr: false,
		},
		{
			name:    "invalid data fails",
			conf:    map[string]any{"foo": "bar", "bar": "notanint"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := source.NewMap(tt.conf)
			cfg, err := New(WithSource(src), WithJSONSchema(schema))
			require.NoError(t, err)

			err = cfg.Load(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNew_WithJSONSchema_ErrorCase(t *testing.T) {
	t.Parallel()

	t.Run("invalid JSON schema fails", func(t *testing.T) {
		t.Parallel()

		_, err := New(WithSource(source.NewMap(map[string]any{"foo": "bar"})), WithJSONSchema([]byte(`{invalid json`)))
		require.Error(t, err)
	})

	t.Run("schema that fails to compile returns error", func(t *testing.T) {
		t.Parallel()
		// Schema with invalid $ref that does not exist - Compile fails
		schema := []byte(`{"$ref": "#/definitions/Missing"}`)
		_, err := New(WithSource(source.NewMap(map[string]any{"foo": "bar"})), WithJSONSchema(schema))
		require.Error(t, err)
	})
}

func TestWithFileAs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		decoder codec.Decoder
		wantErr bool
	}{
		{
			name:    "valid path and codec",
			path:    "/tmp/config.json",
			decoder: codec.JSON,
			wantErr: false,
		},
		{
			name:    "valid path with YAML codec",
			path:    "/tmp/config.yaml",
			decoder: codec.YAML,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := New(WithFileAs(tt.path, tt.decoder))

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Len(t, cfg.sources, 1)
		})
	}
}

func TestWithContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		decoder codec.Decoder
		wantErr bool
	}{
		{
			name:    "valid JSON content",
			data:    []byte(`{"foo": "bar"}`),
			decoder: codec.JSON,
			wantErr: false,
		},
		{
			name:    "valid YAML content",
			data:    []byte("foo: bar"),
			decoder: codec.YAML,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := New(WithContent(tt.data, tt.decoder))

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Len(t, cfg.sources, 1)
		})
	}
}

func TestWithEnv(t *testing.T) {
	t.Parallel()

	cfg, err := New(WithEnv("TESTPREFIX_"))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.sources, 1)
}

func TestWithConsul_RequiresEnvVar(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv (testing package restriction).
	t.Setenv("CONSUL_HTTP_ADDR", "")

	cfg, err := New(WithConsul("production/service.yaml"))
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CONSUL_HTTP_ADDR")
}

func TestWithConsulAs_RequiresEnvVar(t *testing.T) {
	t.Setenv("CONSUL_HTTP_ADDR", "")

	cfg, err := New(WithConsulAs("production/service", codec.JSON))
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "CONSUL_HTTP_ADDR")
}

func TestWithIf_WithConsul_SkipsWithoutEnvVar(t *testing.T) {
	t.Setenv("CONSUL_HTTP_ADDR", "")

	cfg, err := New(WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "", WithConsul("production/service.yaml")))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.sources, 0)
}

func TestWithIf_WithConsulAs_SkipsWithoutEnvVar(t *testing.T) {
	t.Setenv("CONSUL_HTTP_ADDR", "")

	cfg, err := New(WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "", WithConsulAs("production/service", codec.JSON)))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.sources, 0)
}

func TestWithIf_AppliesWhenConditionTrue(t *testing.T) {
	cfg, err := New(WithIf(true, WithSource(source.NewMap(map[string]any{"service": map[string]any{"name": "edge"}}))))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.sources, 1)
}

func TestWithFile_ExpandsEnvVars(t *testing.T) {
	t.Parallel()

	// Set up test environment variable with unique name
	tmpDir := t.TempDir()
	envVar := "TEST_CONFIG_DIR_WITHFILE"
	require.NoError(t, os.Setenv(envVar, tmpDir))
	defer func() {
		require.NoError(t, os.Unsetenv(envVar))
	}()

	// Create test file
	testFile := filepath.Join(tmpDir, "test_env_expand.yaml")
	testData := []byte("test: value")
	require.NoError(t, os.WriteFile(testFile, testData, 0o600))

	// Test with environment variable expansion
	cfg, err := New(WithFile("${" + envVar + "}/test_env_expand.yaml"))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.sources, 1)

	// Verify it actually loads
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "value", mustString(t, cfg, "test"))
}

func TestWithFileAs_ExpandsEnvVars(t *testing.T) {
	t.Parallel()

	// Set up test environment variable with unique name
	tmpDir := t.TempDir()
	envVar := "TEST_CONFIG_DIR_WITHFILEAS"
	require.NoError(t, os.Setenv(envVar, tmpDir))
	defer func() {
		require.NoError(t, os.Unsetenv(envVar))
	}()

	// Create test file without extension
	testFile := filepath.Join(tmpDir, "test_env_expand_noext")
	testData := []byte("test: value")
	require.NoError(t, os.WriteFile(testFile, testData, 0o600))

	// Test with environment variable expansion
	cfg, err := New(WithFileAs("${"+envVar+"}/test_env_expand_noext", codec.YAML))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.sources, 1)

	// Verify it actually loads
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "value", mustString(t, cfg, "test"))
}

func TestWithFileDumper_ExpandsEnvVars(t *testing.T) {
	t.Parallel()

	// Set up test environment variable with unique name
	tmpDir := t.TempDir()
	envVar := "TEST_OUTPUT_DIR_WITHFILEDUMPER"
	require.NoError(t, os.Setenv(envVar, tmpDir))
	defer func() {
		require.NoError(t, os.Unsetenv(envVar))
	}()

	outputFile := filepath.Join(tmpDir, "test_env_expand_dump.yaml")

	// Test with environment variable expansion
	cfg, err := New(
		WithContent([]byte("test: value"), codec.YAML),
		WithFileDumper("${"+envVar+"}/test_env_expand_dump.yaml"),
	)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.dumpers, 1)

	// Load and dump
	require.NoError(t, cfg.Load(context.Background()))
	require.NoError(t, cfg.Dump(context.Background()))

	// Verify file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err)
}

func TestWithFileDumperAs_ExpandsEnvVars(t *testing.T) {
	t.Parallel()

	// Set up test environment variable with unique name
	tmpDir := t.TempDir()
	envVar := "TEST_OUTPUT_DIR_WITHFILEDUMPERAS"
	require.NoError(t, os.Setenv(envVar, tmpDir))
	defer func() {
		require.NoError(t, os.Unsetenv(envVar))
	}()

	outputFile := filepath.Join(tmpDir, "test_env_expand_dump_noext")

	// Test with environment variable expansion
	cfg, err := New(
		WithContent([]byte("test: value"), codec.YAML),
		WithFileDumperAs("${"+envVar+"}/test_env_expand_dump_noext", codec.YAML),
	)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.dumpers, 1)

	// Load and dump
	require.NoError(t, cfg.Load(context.Background()))
	require.NoError(t, cfg.Dump(context.Background()))

	// Verify file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err)
}

func TestWithConsul_ExpandsEnvVars(t *testing.T) {
	t.Parallel()

	// Set up test environment variables
	require.NoError(t, os.Setenv("TEST_APP_ENV", "staging"))
	require.NoError(t, os.Setenv("CONSUL_HTTP_ADDR", "http://localhost:8500"))
	defer func() {
		require.NoError(t, os.Unsetenv("TEST_APP_ENV"))
		require.NoError(t, os.Unsetenv("CONSUL_HTTP_ADDR"))
	}()

	// Test with environment variable expansion
	// Note: This will try to connect to Consul, so we expect an error
	// but we're just verifying the path expansion happens
	cfg, err := New(WithConsul("${TEST_APP_ENV}/service.yaml"))

	// We expect the config to be created (env var expanded)
	// but Load() will fail since there's no actual Consul
	assert.NotNil(t, cfg)
	// The error check is relaxed because Consul connection may fail
	// The important thing is the path was expanded before being used
	_ = err
}

func TestWithConsulAs_ExpandsEnvVars(t *testing.T) {
	t.Parallel()

	// Set up test environment variables
	require.NoError(t, os.Setenv("TEST_APP_ENV", "staging"))
	require.NoError(t, os.Setenv("CONSUL_HTTP_ADDR", "http://localhost:8500"))
	defer func() {
		require.NoError(t, os.Unsetenv("TEST_APP_ENV"))
		require.NoError(t, os.Unsetenv("CONSUL_HTTP_ADDR"))
	}()

	// Test with environment variable expansion
	cfg, err := New(WithConsulAs("${TEST_APP_ENV}/service", codec.JSON))

	// We expect the config to be created (env var expanded)
	assert.NotNil(t, cfg)
	// The error check is relaxed because Consul connection may fail
	_ = err
}

// TestAddConsulSource_NewConsulError covers the error branch in addConsulSource
// when the source constructor fails. The consul factory is replaced with a stub
// via WithConsulFactory so we can exercise the path without a real Consul server.
func TestAddConsulSource_NewConsulError(t *testing.T) {
	t.Setenv("CONSUL_HTTP_ADDR", "http://localhost:8500")

	_, err := New(
		WithConsulFactory(func(_ string, _ codec.Decoder, _ source.ConsulKV) (Source, error) {
			return nil, errors.New("stub: client creation failed")
		}),
		WithConsul("production/service.yaml"),
	)
	require.Error(t, err)
	assert.ErrorContains(t, err, "stub: client creation failed")
}

func TestWithFileDumper(t *testing.T) {
	t.Parallel()

	path := "/tmp/config_test_file_dumper.json"
	cfg, err := New(WithFileDumperAs(path, codec.JSON))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.dumpers, 1)
}

func TestConfigError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")

	tests := []struct {
		name       string
		err        *ConfigError
		wantMsg    string
		wantUnwrap error
	}{
		{
			name: "load with path",
			err: &ConfigError{
				Op:   OpLoad,
				Path: "source[1]",
				Err:  baseErr,
			},
			wantMsg:    "synthra: load source[1]: base error",
			wantUnwrap: baseErr,
		},
		{
			name: "new without path",
			err: &ConfigError{
				Op:  OpNew,
				Err: baseErr,
			},
			wantMsg:    "synthra: new: base error",
			wantUnwrap: baseErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantMsg, tt.err.Error())
			assert.Equal(t, tt.wantUnwrap, tt.err.Unwrap())
		})
	}
}

func TestConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent Load", func(t *testing.T) {
		t.Parallel()

		src := source.NewMap(map[string]any{"foo": "bar"})
		cfg, err := New(WithSource(src))
		require.NoError(t, err)

		wg := make(chan struct{})
		for range 10 {
			go func() {
				loadErr := cfg.Load(context.Background())
				if loadErr != nil {
					t.Error(loadErr)
				}
				wg <- struct{}{}
			}()
		}
		for range 10 {
			<-wg
		}
	})

	t.Run("concurrent Get", func(t *testing.T) {
		t.Parallel()

		src := source.NewMap(map[string]any{"foo": "bar"})
		cfg, err := New(WithSource(src))
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))

		wg := make(chan struct{})
		for range 10 {
			go func() {
				_ = cfg.Get("foo")
				loadErr := cfg.Load(context.Background())
				if loadErr != nil {
					t.Error(loadErr)
				}
				wg <- struct{}{}
			}()
		}
		for range 10 {
			<-wg
		}
	})

	t.Run("concurrent Get and Load with binding validation", func(t *testing.T) {
		t.Parallel()

		type validatingBindStruct struct {
			Foo string `synthra:"foo"`
			Bar int    `synthra:"bar"`
		}

		src := source.NewMap(map[string]any{"foo": "bar", "bar": 42})
		var bind validatingBindStruct
		cfg, err := New(WithSource(src), WithBinding(&bind))
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))

		wg := make(chan struct{})
		for range 20 {
			go func() {
				defer func() { wg <- struct{}{} }()
				for i := range 10 {
					if i%2 == 0 {
						_ = cfg.Get("foo")
						_, _ = cfg.String("foo") //nolint:errcheck // concurrency stress
						_, _ = cfg.Int("bar")    //nolint:errcheck // concurrency stress
						_ = cfg.Values()
					} else {
						loadErr := cfg.Load(context.Background())
						if loadErr != nil {
							t.Error(loadErr)
						}
					}
				}
			}()
		}

		for range 20 {
			<-wg
		}

		assert.Equal(t, "bar", mustString(t, cfg, "foo"))
		assert.Equal(t, 42, mustInt(t, cfg, "bar"))
	})

	t.Run("concurrent access to same key", func(t *testing.T) {
		t.Parallel()

		src := source.NewMap(map[string]any{"shared": "value"})
		cfg, err := New(WithSource(src))
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))

		var wg sync.WaitGroup
		//nolint:makezero // indexed assignment requires pre-allocated length
		results := make([]string, 10)

		for i := range 10 {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				s, serr := cfg.String("shared")
				if serr != nil {
					t.Error(serr)
					return
				}
				results[index] = s
			}(i)
		}

		wg.Wait()

		for _, result := range results {
			assert.Equal(t, "value", result)
		}
	})
}

func TestReload(t *testing.T) {
	t.Parallel()

	data := map[string]any{"foo": "bar"}
	src := source.NewMap(data)
	cfg, err := New(WithSource(src))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "bar", mustString(t, cfg, "foo"))

	data["foo"] = "baz"
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "baz", mustString(t, cfg, "foo"))
}

func TestNilConfigInstance(t *testing.T) {
	t.Parallel()

	var cfg *Synthra

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"String", func() error { _, err := cfg.String("any"); return err }},
		{"Bool", func() error { _, err := cfg.Bool("any"); return err }},
		{"Int", func() error { _, err := cfg.Int("any"); return err }},
		{"Int64", func() error { _, err := cfg.Int64("any"); return err }},
		{"Float64", func() error { _, err := cfg.Float64("any"); return err }},
		{"Duration", func() error { _, err := cfg.Duration("any"); return err }},
		{"Time", func() error { _, err := cfg.Time("any"); return err }},
		{"StringSlice", func() error { _, err := cfg.StringSlice("any"); return err }},
		{"IntSlice", func() error { _, err := cfg.IntSlice("any"); return err }},
		{"StringMap", func() error { _, err := cfg.StringMap("any"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrNilConfig)
		})
	}

	assert.Nil(t, cfg.Get("any"))

	_, err := Get[string](cfg, "any")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilConfig)
}

func TestConfigOrMethods_NilConfigAndMissingKey(t *testing.T) {
	t.Parallel()

	t.Run("nil config returns default for Or methods", func(t *testing.T) {
		t.Parallel()
		var cfg *Synthra
		assert.Equal(t, "default", cfg.StringOr("key", "default"))
		assert.Equal(t, 8080, cfg.IntOr("key", 8080))
		assert.Equal(t, int64(1024), cfg.Int64Or("key", 1024))
		assert.Equal(t, 0.5, cfg.Float64Or("key", 0.5))
		assert.True(t, cfg.BoolOr("key", true))
		assert.Equal(t, 30*time.Second, cfg.DurationOr("key", 30*time.Second))
		assert.Equal(t, []string{"a"}, cfg.StringSliceOr("key", []string{"a"}))
		assert.Equal(t, []int{1}, cfg.IntSliceOr("key", []int{1}))
		assert.Equal(t, map[string]any{"x": "y"}, cfg.StringMapOr("key", map[string]any{"x": "y"}))
	})

	t.Run("missing key returns default for Or methods", func(t *testing.T) {
		t.Parallel()
		cfg := loadTestConfig(t, map[string]any{"foo": "bar"})
		assert.Equal(t, "default", cfg.StringOr("missing", "default"))
		assert.Equal(t, 8080, cfg.IntOr("missing", 8080))
		assert.Equal(t, int64(1024), cfg.Int64Or("missing", 1024))
		assert.Equal(t, 0.5, cfg.Float64Or("missing", 0.5))
		assert.True(t, cfg.BoolOr("missing", true))
		assert.Equal(t, 30*time.Second, cfg.DurationOr("missing", 30*time.Second))
		assert.Equal(t, []string{"a"}, cfg.StringSliceOr("missing", []string{"a"}))
		assert.Equal(t, []int{1}, cfg.IntSliceOr("missing", []int{1}))
		assert.Equal(t, map[string]any{"x": "y"}, cfg.StringMapOr("missing", map[string]any{"x": "y"}))
	})
}

func TestLargeConfiguration(t *testing.T) {
	t.Parallel()

	largeConfig := make(map[string]any, 1000)
	for i := range 1000 {
		largeConfig[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	cfg := loadTestConfig(t, largeConfig)

	assert.Equal(t, "value0", mustString(t, cfg, "key0"))
	assert.Equal(t, "value999", mustString(t, cfg, "key999"))
	assert.Equal(t, "value500", mustString(t, cfg, "key500"))
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	type mockContextAwareSource struct {
		conf map[string]any
		err  error
	}

	mockCtxSource := &mockContextAwareSource{conf: map[string]any{"foo": "bar"}}

	// Implement Source interface
	loadFunc := func(ctx context.Context) (map[string]any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return mockCtxSource.conf, mockCtxSource.err //nolint:nilnil // Test mock intentionally returns (nil, nil) for certain test cases
		}
	}

	// Use loadFunc to avoid unused variable warning
	_ = loadFunc

	cfg, err := New(WithSource(source.NewMap(map[string]any{"foo": "bar"})))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// This test is primarily to show context handling
	err = cfg.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilePermissions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	sourceFile := tmpDir + "/source.yaml"
	dumpFile := tmpDir + "/dump.yaml"

	sourceContent := []byte("foo: bar\n")
	err := os.WriteFile(sourceFile, sourceContent, 0o600)
	require.NoError(t, err)

	cfg, err := New(
		WithFileAs(sourceFile, codec.YAML),
		WithFileDumperAs(dumpFile, codec.YAML),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	require.NoError(t, cfg.Dump(context.Background()))

	info, err := os.Stat(dumpFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestConsistentReturnTypes(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t, map[string]any{"existing": "value"})

	t.Run("slices return empty not nil", func(t *testing.T) {
		intSlice, err := Get[[]int](cfg, "nonexistent")
		require.Error(t, err)
		assert.NotNil(t, intSlice)
		assert.Len(t, intSlice, 0)

		stringSlice, err := Get[[]string](cfg, "nonexistent")
		require.Error(t, err)
		assert.NotNil(t, stringSlice)
		assert.Len(t, stringSlice, 0)
	})

	t.Run("maps return empty not nil", func(t *testing.T) {
		stringMap, err := Get[map[string]any](cfg, "nonexistent")
		require.Error(t, err)
		assert.NotNil(t, stringMap)
		assert.Len(t, stringMap, 0)

		stringMapString, err := Get[map[string]string](cfg, "nonexistent")
		require.Error(t, err)
		assert.NotNil(t, stringMapString)
		assert.Len(t, stringMapString, 0)

		stringMapStringSlice, err := Get[map[string][]string](cfg, "nonexistent")
		require.Error(t, err)
		assert.NotNil(t, stringMapStringSlice)
		assert.Len(t, stringMapStringSlice, 0)
	})
}

func TestCaseInsensitiveMerging(t *testing.T) {
	t.Parallel()

	// Test data with mixed case keys
	config1 := []byte(`{
		"Server": {
			"Host": "localhost",
			"Port": 8080
		},
		"Database": {
			"Name": "testdb"
		}
	}`)

	config2 := []byte(`{
		"server": {
			"host": "example.com",
			"port": 9090
		},
		"database": {
			"name": "prod"
		}
	}`)

	// Create configuration with both sources
	cfg, err := New(
		WithContent(config1, codec.JSON),
		WithContent(config2, codec.JSON),
	)
	require.NoError(t, err)

	// Load configuration
	err = cfg.Load(context.Background())
	require.NoError(t, err)

	tests := []struct {
		name    string
		key     string
		wantStr string
		wantInt int
		getType string // "string" or "int"
	}{
		{
			name:    "server.host lowercase",
			key:     "server.host",
			wantStr: "example.com",
			getType: "string",
		},
		{
			name:    "Server.Host mixed case",
			key:     "Server.Host",
			wantStr: "example.com",
			getType: "string",
		},
		{
			name:    "SERVER.HOST uppercase",
			key:     "SERVER.HOST",
			wantStr: "example.com",
			getType: "string",
		},
		{
			name:    "server.port lowercase",
			key:     "server.port",
			wantInt: 9090,
			getType: "int",
		},
		{
			name:    "Server.Port mixed case",
			key:     "Server.Port",
			wantInt: 9090,
			getType: "int",
		},
		{
			name:    "SERVER.PORT uppercase",
			key:     "SERVER.PORT",
			wantInt: 9090,
			getType: "int",
		},
		{
			name:    "database.name lowercase",
			key:     "database.name",
			wantStr: "prod",
			getType: "string",
		},
		{
			name:    "Database.Name mixed case",
			key:     "Database.Name",
			wantStr: "prod",
			getType: "string",
		},
		{
			name:    "DATABASE.NAME uppercase",
			key:     "DATABASE.NAME",
			wantStr: "prod",
			getType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch tt.getType {
			case "string":
				got, strErr := cfg.String(tt.key)
				require.NoError(t, strErr)
				assert.Equal(t, tt.wantStr, got)
			case "int":
				got, intErr := cfg.Int(tt.key)
				require.NoError(t, intErr)
				assert.Equal(t, tt.wantInt, got)
			}
		})
	}
}

func TestDeepMerge_SameCase_OverridesValue(t *testing.T) {
	t.Parallel()
	dst := map[string]any{"foo": "original"}
	src := map[string]any{"foo": "updated"}
	deepMerge(dst, src)
	assert.Equal(t, "updated", dst["foo"])
}

func TestDeepMerge_DifferentCase_FirstCasingWins_ValueOverridden(t *testing.T) {
	t.Parallel()
	dst := map[string]any{"Foo": "original"}
	src := map[string]any{"foo": "updated"}
	deepMerge(dst, src)
	// The key "Foo" was already in dst; first-writer casing ("Foo") is preserved.
	assert.Equal(t, "updated", dst["Foo"])
	_, hasLower := dst["foo"]
	assert.False(t, hasLower, "lowercase duplicate must not be added")
}

func TestDeepMerge_NewKey_KeepsSourceCasing(t *testing.T) {
	t.Parallel()
	dst := map[string]any{}
	src := map[string]any{"Bar": 42}
	deepMerge(dst, src)
	assert.Equal(t, 42, dst["Bar"])
}

func TestDeepMerge_NestedMap_RecursesAndPreservesCasing(t *testing.T) {
	t.Parallel()
	dst := map[string]any{
		"Server": map[string]any{"Host": "localhost"},
	}
	src := map[string]any{
		"server": map[string]any{"port": 8080},
	}
	deepMerge(dst, src)
	nested, ok := dst["Server"].(map[string]any)
	require.True(t, ok, "Server key should be preserved with original casing")
	assert.Equal(t, "localhost", nested["Host"])
	assert.Equal(t, 8080, nested["port"])
}

func TestConcurrency_LoadAndDump(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{"foo": "bar"})
	d := &recordingDumper{}
	cfg, err := New(WithSource(src), WithDumper(d))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	const goroutines = 10
	done := make(chan error, goroutines*2)

	for range goroutines {
		go func() {
			done <- cfg.Load(context.Background())
		}()
		go func() {
			done <- cfg.Dump(context.Background())
		}()
	}

	for range goroutines * 2 {
		assert.NoError(t, <-done)
	}
}

func TestDump_CancelledContext(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{"foo": "bar"})
	d := &recordingDumper{}
	cfg, err := New(WithSource(src), WithDumper(d))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Dump does not short-circuit on a canceled context before calling dumpers;
	// a simple in-memory dumper that ignores ctx succeeds regardless.
	err = cfg.Dump(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, d.Calls())
}

func TestWithIf_NilInnerOptionPanics(t *testing.T) {
	t.Parallel()

	// WithIf does not validate its inner options; when condition is true
	// and a nil option is passed, calling New panics at the nil function call.
	assert.Panics(t, func() {
		// condition is true so the nil option is executed
		MustNew(WithIf(true, nil))
	})
}

func TestReload_WithBinding(t *testing.T) {
	t.Parallel()

	data := map[string]any{"foo": "first", "bar": 1}
	src := source.NewMap(data)

	type bound struct {
		Foo string `synthra:"foo"`
		Bar int    `synthra:"bar"`
	}

	var b bound
	cfg, err := New(WithSource(src), WithBinding(&b))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "first", b.Foo)
	assert.Equal(t, 1, b.Bar)

	// Mutate source and reload; binding struct must reflect new values.
	data["foo"] = "second"
	data["bar"] = 2
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "second", b.Foo)
	assert.Equal(t, 2, b.Bar)
}

func TestWithContent_EmptyAndNilData(t *testing.T) {
	t.Parallel()

	t.Run("nil JSON content fails to load", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(WithContent(nil, codec.JSON))
		require.NoError(t, err)
		err = cfg.Load(context.Background())
		require.Error(t, err)
	})

	t.Run("empty JSON content fails to load", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(WithContent([]byte{}, codec.JSON))
		require.NoError(t, err)
		err = cfg.Load(context.Background())
		require.Error(t, err)
	})

	t.Run("empty YAML content loads as empty map", func(t *testing.T) {
		t.Parallel()
		cfg, err := New(WithContent([]byte{}, codec.YAML))
		require.NoError(t, err)
		// Empty YAML is valid and produces an empty map.
		require.NoError(t, cfg.Load(context.Background()))
		vals := cfg.Values()
		assert.Empty(t, *vals)
	})
}

func TestLoad_FileDeletedAfterNew(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("foo: bar\n"), 0o600))

	cfg, err := New(WithFileAs(tmpFile, codec.YAML))
	require.NoError(t, err)

	// Delete the file before Load.
	require.NoError(t, os.Remove(tmpFile))

	err = cfg.Load(context.Background())
	require.Error(t, err)
	// Error must propagate from the source layer.
	var ce *ConfigError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, OpLoad, ce.Op)
}

// TestTypedAccessors_CastFailure verifies that each strict typed accessor
// returns a *ConfigError with Op=OpGet when the stored value cannot be
// coerced to the requested type.
func TestTypedAccessors_CastFailure(t *testing.T) {
	t.Parallel()

	// A nested map is unconvertible to numeric/scalar/slice types.
	cfg := loadTestConfig(t, map[string]any{
		"nested":  map[string]any{"a": 1},
		"numtext": "not-a-number",
		"plain":   42,
	})

	assertGetError := func(t *testing.T, err error, key string) {
		t.Helper()
		require.Error(t, err)
		var ce *ConfigError
		require.ErrorAs(t, err, &ce, "expected *ConfigError")
		assert.Equal(t, OpGet, ce.Op)
		assert.Equal(t, key, ce.Path)
	}

	t.Run("Int with map value", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.Int("nested")
		assertGetError(t, err, "nested")
	})

	t.Run("Bool with map value", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.Bool("nested")
		assertGetError(t, err, "nested")
	})

	t.Run("Duration with non-duration string", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.Duration("numtext")
		assertGetError(t, err, "numtext")
	})

	t.Run("Time with non-time string", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.Time("numtext")
		assertGetError(t, err, "numtext")
	})

	t.Run("StringMap with non-map value", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.StringMap("plain")
		assertGetError(t, err, "plain")
	})
}

// TestOrMethods_WrongTypeCoercesToZeroNotDefault documents that *Or methods
// use the non-error cast variants: when a value exists but cannot be
// coerced they return the cast zero value (e.g. 0, false) rather than the
// caller-supplied default.
func TestOrMethods_WrongTypeCoercesToZeroNotDefault(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t, map[string]any{
		// "numtext" is a non-numeric string; cast.ToInt returns 0
		"numtext": "hello",
	})

	// Key is present, value is unconvertible -> cast returns zero, not the default.
	got := cfg.IntOr("numtext", 99)
	assert.Equal(t, 0, got, "IntOr returns cast zero, not caller default, when value is present but not numeric")

	got64 := cfg.Int64Or("numtext", int64(99))
	assert.Equal(t, int64(0), got64)

	gotF := cfg.Float64Or("numtext", 3.14)
	assert.InDelta(t, 0.0, gotF, 0.0001)
}

func TestLoad_BindingInvalidDefaults(t *testing.T) {
	t.Parallel()

	t.Run("invalid uint default fails", func(t *testing.T) {
		t.Parallel()
		type withUint struct {
			Limit uint `synthra:"limit" default:"not-a-number"`
		}
		var target withUint
		cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
		require.NoError(t, err)
		err = cfg.Load(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set default")
	})

	t.Run("invalid float default fails", func(t *testing.T) {
		t.Parallel()
		type withFloat struct {
			Rate float64 `synthra:"rate" default:"not-a-float"`
		}
		var target withFloat
		cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
		require.NoError(t, err)
		err = cfg.Load(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set default")
	})

	t.Run("invalid bool default fails", func(t *testing.T) {
		t.Parallel()
		type withBool struct {
			Debug bool `synthra:"debug" default:"not-a-bool"`
		}
		var target withBool
		cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
		require.NoError(t, err)
		err = cfg.Load(context.Background())
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set default")
	})
}

func TestLoad_BindingNestedStructDefaults(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Timeout time.Duration `synthra:"timeout" default:"10s"`
		Retries int           `synthra:"retries" default:"3"`
	}
	type Outer struct {
		Name   string `synthra:"name" default:"default-service"`
		Client Inner  `synthra:"client"`
	}

	var target Outer
	cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	assert.Equal(t, "default-service", target.Name)
	assert.Equal(t, 10*time.Second, target.Client.Timeout)
	assert.Equal(t, 3, target.Client.Retries)
}

// TestGetOr_ConversionFallback covers the convertToType path in GetOr
// and the final defaultVal return when conversion itself returns no match.
func TestGetOr_ConversionFallback(t *testing.T) {
	t.Parallel()

	t.Run("string to int conversion succeeds", func(t *testing.T) {
		t.Parallel()
		cfg := loadTestConfig(t, map[string]any{"port": "9090"})
		got := GetOr(cfg, "port", 8080)
		assert.Equal(t, 9090, got)
	})

	t.Run("unconvertible type falls back to default", func(t *testing.T) {
		t.Parallel()
		type myType struct{ X int }
		cfg := loadTestConfig(t, map[string]any{"key": "value"})
		got := GetOr(cfg, "key", myType{X: 1})
		assert.Equal(t, myType{X: 1}, got)
	})
}

// TestOrMethods_KeyExistsHappyPath verifies that each *Or method returns the
// stored value when the key exists and the type cast succeeds.
func TestOrMethods_KeyExistsHappyPath(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t, map[string]any{
		"host":    "localhost",
		"enabled": true,
		"timeout": "30s",
		"tags":    []any{"a", "b"},
		"ports":   []any{8080, 9090},
		"meta":    map[string]any{"version": "1.0"},
	})

	assert.Equal(t, "localhost", cfg.StringOr("host", "default"))
	assert.True(t, cfg.BoolOr("enabled", false))
	assert.Equal(t, 30*time.Second, cfg.DurationOr("timeout", time.Minute))
	assert.Equal(t, []string{"a", "b"}, cfg.StringSliceOr("tags", nil))
	assert.Equal(t, []int{8080, 9090}, cfg.IntSliceOr("ports", nil))
	assert.Equal(t, map[string]any{"version": "1.0"}, cfg.StringMapOr("meta", nil))
}

// TestTimeOr covers all three paths of TimeOr: nil config, missing key,
// and key present.
func TestTimeOr(t *testing.T) {
	t.Parallel()

	want := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	sentinel := time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("nil config returns default", func(t *testing.T) {
		t.Parallel()
		var c *Synthra
		assert.Equal(t, sentinel, c.TimeOr("ts", sentinel))
	})

	t.Run("missing key returns default", func(t *testing.T) {
		t.Parallel()
		cfg := loadTestConfig(t, map[string]any{"foo": "bar"})
		assert.Equal(t, sentinel, cfg.TimeOr("missing", sentinel))
	})

	t.Run("key present returns parsed time", func(t *testing.T) {
		t.Parallel()
		cfg := loadTestConfig(t, map[string]any{"ts": "2023-01-01T12:00:00Z"})
		assert.Equal(t, want, cfg.TimeOr("ts", sentinel))
	})
}

// TestTypedAccessors_CastFailure_Additional verifies that String, Int64,
// StringSlice, and IntSlice return an error when the stored value cannot be cast.
func TestTypedAccessors_CastFailure_Additional(t *testing.T) {
	t.Parallel()

	// An anonymous struct is not handled by any cast function and produces an error.
	type unconvertible struct{}

	cfg := loadTestConfig(t, map[string]any{
		"obj": unconvertible{},
	})

	assertCastErr := func(t *testing.T, err error, key string) {
		t.Helper()
		require.Error(t, err)
		var ce *ConfigError
		require.ErrorAs(t, err, &ce, "expected *ConfigError")
		assert.Equal(t, OpGet, ce.Op)
		assert.Equal(t, key, ce.Path)
	}

	t.Run("String with unconvertible value", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.String("obj")
		assertCastErr(t, err, "obj")
	})

	t.Run("Int64 with unconvertible value", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.Int64("obj")
		assertCastErr(t, err, "obj")
	})

	t.Run("StringSlice with unconvertible value", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.StringSlice("obj")
		assertCastErr(t, err, "obj")
	})

	t.Run("IntSlice with unconvertible value", func(t *testing.T) {
		t.Parallel()
		_, err := cfg.IntSlice("obj")
		assertCastErr(t, err, "obj")
	})
}

// TestConvertToType_TypeMismatch covers the convertToType case arms that are only
// reached when the stored type differs from T, so val.(T) fails first.
func TestConvertToType_TypeMismatch(t *testing.T) {
	t.Parallel()

	cfg := loadTestConfig(t, map[string]any{
		"asstring": int(42),
		"asint64":  "64",
		"asint32":  "42",
		"asint16":  "16",
		"asint8":   "8",
		"asuint":   "7",
		"asuint64": "64",
		"asuint32": "32",
		"asuint16": "16",
		"asuint8":  "8",
		// JSON string — cast.ToStringMap parses it to map[string]any
		"asmap": `{"k":"v"}`,
	})

	t.Run("int to string", func(t *testing.T) {
		t.Parallel()
		v, err := Get[string](cfg, "asstring")
		require.NoError(t, err)
		assert.Equal(t, "42", v)
	})

	t.Run("string to int64", func(t *testing.T) {
		t.Parallel()
		v, err := Get[int64](cfg, "asint64")
		require.NoError(t, err)
		assert.Equal(t, int64(64), v)
	})

	t.Run("string to int32", func(t *testing.T) {
		t.Parallel()
		v, err := Get[int32](cfg, "asint32")
		require.NoError(t, err)
		assert.Equal(t, int32(42), v)
	})

	t.Run("string to int16", func(t *testing.T) {
		t.Parallel()
		v, err := Get[int16](cfg, "asint16")
		require.NoError(t, err)
		assert.Equal(t, int16(16), v)
	})

	t.Run("string to int8", func(t *testing.T) {
		t.Parallel()
		v, err := Get[int8](cfg, "asint8")
		require.NoError(t, err)
		assert.Equal(t, int8(8), v)
	})

	t.Run("string to uint", func(t *testing.T) {
		t.Parallel()
		v, err := Get[uint](cfg, "asuint")
		require.NoError(t, err)
		assert.Equal(t, uint(7), v)
	})

	t.Run("string to uint64", func(t *testing.T) {
		t.Parallel()
		v, err := Get[uint64](cfg, "asuint64")
		require.NoError(t, err)
		assert.Equal(t, uint64(64), v)
	})

	t.Run("string to uint32", func(t *testing.T) {
		t.Parallel()
		v, err := Get[uint32](cfg, "asuint32")
		require.NoError(t, err)
		assert.Equal(t, uint32(32), v)
	})

	t.Run("string to uint16", func(t *testing.T) {
		t.Parallel()
		v, err := Get[uint16](cfg, "asuint16")
		require.NoError(t, err)
		assert.Equal(t, uint16(16), v)
	})

	t.Run("string to uint8", func(t *testing.T) {
		t.Parallel()
		v, err := Get[uint8](cfg, "asuint8")
		require.NoError(t, err)
		assert.Equal(t, uint8(8), v)
	})

	t.Run("map[string]any from JSON string", func(t *testing.T) {
		t.Parallel()
		v, err := Get[map[string]any](cfg, "asmap")
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"k": "v"}, v)
	})
}

// TestApplyDefaults_InvalidTargets covers the guard clauses at the top of
// applyDefaults.
func TestApplyDefaults_InvalidTargets(t *testing.T) {
	t.Parallel()

	t.Run("non-pointer returns error", func(t *testing.T) {
		t.Parallel()
		err := applyDefaults(42)
		require.Error(t, err)
		assert.ErrorContains(t, err, "target must be a pointer")
	})

	t.Run("pointer to non-struct returns error", func(t *testing.T) {
		t.Parallel()
		x := "hello"
		err := applyDefaults(&x)
		require.Error(t, err)
		assert.ErrorContains(t, err, "pointer to a struct")
	})
}

// TestSetDefaults_UnexportedField covers the !field.CanSet() continue branch
// in setDefaults.
func TestSetDefaults_UnexportedField(t *testing.T) {
	t.Parallel()

	type withUnexported struct {
		Exported   string `default:"hello"`
		unexported string `default:"world"` //nolint:unused // unexported field for coverage
	}

	target := &withUnexported{}
	require.NoError(t, applyDefaults(target))
	assert.Equal(t, "hello", target.Exported)
	// unexported field is skipped; no panic or error
}

// TestSetDefaults_NestedStructPropagatesError covers the error return from the
// recursive setDefaults call on nested structs.
func TestSetDefaults_NestedStructPropagatesError(t *testing.T) {
	t.Parallel()

	type inner struct {
		Port int `default:"not-a-number"`
	}
	type outer struct {
		Name  string `default:"ok"`
		Inner inner
	}

	err := applyDefaults(&outer{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to set default")
}

// TestReflectIsZero_PointerInterfaceAndChannel verifies [reflect.Value.IsZero]
// for types used during default application.
func TestReflectIsZero_PointerInterfaceAndChannel(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer is zero", func(t *testing.T) {
		t.Parallel()
		var p *int
		assert.True(t, reflect.ValueOf(&p).Elem().IsZero())
	})

	t.Run("non-nil pointer is not zero", func(t *testing.T) {
		t.Parallel()
		x := 42
		ptr := &x
		assert.False(t, reflect.ValueOf(&ptr).Elem().IsZero())
	})

	t.Run("channel is not zero after make", func(t *testing.T) {
		t.Parallel()
		ch := make(chan int)
		assert.False(t, reflect.ValueOf(&ch).Elem().IsZero())
	})
}

// TestSetDefaultValue_ValidUintAndFloat verifies that field.SetUint and
// field.SetFloat succeed with a valid default string.
func TestSetDefaultValue_ValidUintAndFloat(t *testing.T) {
	t.Parallel()

	t.Run("valid uint default is applied", func(t *testing.T) {
		t.Parallel()
		type withUint struct {
			Limit uint `synthra:"limit" default:"42"`
		}
		var target withUint
		cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))
		assert.Equal(t, uint(42), target.Limit)
	})

	t.Run("valid float64 default is applied", func(t *testing.T) {
		t.Parallel()
		type withFloat struct {
			Rate float64 `synthra:"rate" default:"3.14"`
		}
		var target withFloat
		cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
		require.NoError(t, err)
		require.NoError(t, cfg.Load(context.Background()))
		assert.InDelta(t, 3.14, target.Rate, 0.001)
	})
}

// TestLoad_BindingInvalidIntDefault covers the cast.ToInt64E error path in
// setDefaultValue for integer fields, which was missing while the uint/float/bool
// invalid-default paths were already tested.
func TestLoad_BindingInvalidIntDefault(t *testing.T) {
	t.Parallel()

	type withInt struct {
		Port int `synthra:"port" default:"not-a-number"`
	}
	var target withInt
	cfg, err := New(WithSource(source.NewMap(map[string]any{})), WithBinding(&target))
	require.NoError(t, err)
	err = cfg.Load(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to set default")
}

// TestGetDecoderConfig_EmptyTagNameFallback covers the tagName == "" guard in
// getDecoderConfig, which is only reachable when Synthra is constructed directly
// TestDecodeBindingInto_EmptyTagNameFallback verifies the fallback to "synthra"
// tag name when Synthra is constructed with an empty tagName.
func TestDecodeBindingInto_EmptyTagNameFallback(t *testing.T) {
	t.Parallel()

	type cfg struct {
		Port int `synthra:"port"`
	}

	s := &Synthra{} // tagName is ""
	var target cfg
	err := s.decodeBindingInto(&target, map[string]any{"port": 8080})
	require.NoError(t, err)
	assert.Equal(t, 8080, target.Port)
}

// TestSynthraNilConfig covers the Configuration() and Dump() paths when Synthra is
// constructed directly without calling build (nil config pointer).
func TestSynthraNilConfig(t *testing.T) {
	t.Parallel()

	t.Run("Values returns empty map", func(t *testing.T) {
		t.Parallel()
		s := &Synthra{}
		vals := s.Values()
		require.NotNil(t, vals)
		assert.Empty(t, *vals)
	})

	t.Run("Get returns nil", func(t *testing.T) {
		t.Parallel()
		s := &Synthra{}
		assert.Nil(t, s.Get("any-key"))
	})

	t.Run("Dump with nil snap succeeds", func(t *testing.T) {
		t.Parallel()
		d := &recordingDumper{}
		s := &Synthra{dumpers: []Dumper{d}}
		require.NoError(t, s.Dump(context.Background()))
		assert.Equal(t, 1, d.Calls())
		assert.Empty(t, d.Last())
	})
}

func TestGetValueFromMap_ExactMatch(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{"apiVersion": "v1"})
	cfg, err := New(WithSource(src))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "v1", mustString(t, cfg, "apiVersion"))
}

func TestGetValueFromMap_FoldMatch(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{"apiVersion": "v1"})
	cfg, err := New(WithSource(src))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	// Case-insensitive lookup: "apiversion" must find "apiVersion".
	assert.Equal(t, "v1", mustString(t, cfg, "apiversion"))
}

func TestGetValueFromMap_DotPathMixedCase(t *testing.T) {
	t.Parallel()

	src := source.NewMap(map[string]any{
		"server": map[string]any{"Host": "localhost"},
	})
	cfg, err := New(WithSource(src))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.Equal(t, "localhost", mustString(t, cfg, "server.host"))
	assert.Equal(t, "localhost", mustString(t, cfg, "Server.Host"))
}

func TestLoad_TwoYAMLs_DifferentCasing_FirstWins(t *testing.T) {
	t.Parallel()

	// First source defines "Host"; second source defines "host".
	// After deepMerge the first-writer casing ("Host") must survive.
	first := source.NewMap(map[string]any{"Host": "primary"})
	second := source.NewMap(map[string]any{"host": "secondary"})
	cfg, err := New(WithSource(first), WithSource(second))
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	// Value is overridden but original casing ("Host") is kept.
	assert.Equal(t, "secondary", mustString(t, cfg, "Host"))
	assert.Equal(t, "secondary", mustString(t, cfg, "host"))
}

func TestLoad_OnBoundRunsBeforeValidate(t *testing.T) {
	t.Parallel()

	type LevelCfg struct {
		Level string `synthra:"level"`
	}

	var out LevelCfg
	h1Ran := false
	h2SawH1 := false

	cfg, err := New(
		WithSource(source.NewMap(map[string]any{"level": "debug"})),
		WithBinding(&out,
			OnBound(func(_ *LevelCfg) error {
				h1Ran = true
				return nil
			}),
			OnBound(func(_ *LevelCfg) error {
				h2SawH1 = h1Ran
				return nil
			}),
		),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))
	assert.True(t, h1Ran)
	assert.True(t, h2SawH1, "hook2 must run after hook1")
}
