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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigError_Error(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")

	tests := []struct {
		name    string
		err     *ConfigError
		wantMsg string
	}{
		{
			name: "with path and err",
			err: &ConfigError{
				Op:   OpLoad,
				Path: "source[0]",
				Err:  baseErr,
			},
			wantMsg: "synthra: load source[0]: base error",
		},
		{
			name: "with path but no err",
			err: &ConfigError{
				Op:   OpLoad,
				Path: "source[0]",
			},
			wantMsg: "synthra: load source[0]",
		},
		{
			name: "without path with err",
			err: &ConfigError{
				Op:  OpLoad,
				Err: ErrNilContext,
			},
			wantMsg: "synthra: load: synthra: nil context",
		},
		{
			name:    "op only, no path, no err",
			err:     &ConfigError{Op: OpNew},
			wantMsg: "synthra: new",
		},
		{
			name:    "nil receiver",
			err:     (*ConfigError)(nil),
			wantMsg: "synthra: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestConfigError_NilReceiver(t *testing.T) {
	t.Parallel()

	var ce *ConfigError
	assert.Nil(t, ce.Unwrap())
}

func TestConfigError_Unwrap(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	ce := NewConfigError(OpGet, "server.port", baseErr)
	assert.Equal(t, baseErr, ce.Unwrap())
}

func TestNewConfigError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	ce := NewConfigError(OpNew, "WithFile", baseErr)
	assert.Equal(t, OpNew, ce.Op)
	assert.Equal(t, "WithFile", ce.Path)
	assert.Equal(t, baseErr, ce.Err)
}

func TestConfigError_IsSentinelsViaUnwrap(t *testing.T) {
	t.Parallel()

	wrappedKey := fmt.Errorf("wrap: %w", ErrKeyNotFound)
	ce := NewConfigError(OpGet, "k", wrappedKey)
	assert.True(t, errors.Is(ce, ErrKeyNotFound))

	ce2 := NewConfigError(OpLoad, "", ErrNilContext)
	assert.True(t, errors.Is(ce2, ErrNilContext))

	ce3 := NewConfigError(OpNew, "WithBinding", ErrNilConfig)
	assert.True(t, errors.Is(ce3, ErrNilConfig))
}

func TestConfigError_ErrorWrapping(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error")

	t.Run("errors_Is_traverses_chain", func(t *testing.T) {
		t.Parallel()
		err := NewConfigError(OpGet, "port", fmt.Errorf("cast: %w", originalErr))
		assert.True(t, errors.Is(err, originalErr))
		var ce *ConfigError
		require.True(t, errors.As(err, &ce))
		assert.Equal(t, OpGet, ce.Op)
	})

	t.Run("errors_As_unwrap", func(t *testing.T) {
		t.Parallel()
		err := NewConfigError(OpLoad, "source[1]", originalErr)
		var ce *ConfigError
		require.True(t, errors.As(err, &ce))
		assert.Equal(t, originalErr, ce.Unwrap())
	})
}

func TestConfigError_Chaining(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error")
	firstErr := NewConfigError(OpLoad, "source[0]", originalErr)
	secondErr := NewConfigError(OpLoad, "binding-decode", firstErr)

	var inner *ConfigError
	require.True(t, errors.As(secondErr, &inner))
	assert.Equal(t, "binding-decode", inner.Path)
	require.True(t, errors.Is(secondErr, originalErr))
}

func TestConfigError_JoinedErrors(t *testing.T) {
	t.Parallel()

	e1 := NewConfigError(OpNew, "WithFile", errors.New("a"))
	e2 := NewConfigError(OpNew, "WithTag", errors.New("b"))
	joined := errors.Join(e1, e2)

	var first *ConfigError
	require.True(t, errors.As(joined, &first))
	assert.Equal(t, OpNew, first.Op)

	unwrapMulti, ok := joined.(interface{ Unwrap() []error })
	require.True(t, ok)
	children := unwrapMulti.Unwrap()
	require.Len(t, children, 2)
	var ce0, ce1 *ConfigError
	require.True(t, errors.As(children[0], &ce0))
	require.True(t, errors.As(children[1], &ce1))
	assert.Equal(t, "WithFile", ce0.Path)
	assert.Equal(t, "WithTag", ce1.Path)
}

func TestSentinelErrors_KeyNotFoundWrapping(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("lookup: %w", ErrKeyNotFound)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}
