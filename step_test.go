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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaStep_Kind(t *testing.T) {
	t.Parallel()
	s := &schemaStep{}
	assert.Equal(t, "schema", s.kind())
}

func TestSchemaStep_Run_ValidSchema(t *testing.T) {
	t.Parallel()

	schema := []byte(`{
		"type": "object",
		"properties": {
			"port": {"type": "integer", "default": 8080},
			"host": {"type": "string"}
		}
	}`)

	s := &schemaStep{
		selector: func(_ map[string]any) ([]byte, error) { return schema, nil },
	}

	got, err := s.run(map[string]any{"host": "localhost"})
	require.NoError(t, err)
	assert.Equal(t, "localhost", got["host"])
	assert.Equal(t, float64(8080), got["port"], "default should be applied")
}

func TestSchemaStep_Run_SelectorError(t *testing.T) {
	t.Parallel()

	selectorErr := errors.New("selector failed")
	s := &schemaStep{
		selector: func(_ map[string]any) ([]byte, error) { return nil, selectorErr },
	}

	_, err := s.run(map[string]any{})
	require.ErrorIs(t, err, selectorErr)
}

func TestSchemaStep_Run_InvalidSchemaBytes(t *testing.T) {
	t.Parallel()

	s := &schemaStep{
		selector: func(_ map[string]any) ([]byte, error) { return []byte(`not valid json`), nil },
	}

	_, err := s.run(map[string]any{})
	require.Error(t, err)
}

func TestSchemaStep_Run_ValidationFailure(t *testing.T) {
	t.Parallel()

	schema := []byte(`{
		"type": "object",
		"required": ["name"],
		"properties": {
			"name": {"type": "string"}
		}
	}`)

	s := &schemaStep{
		selector: func(_ map[string]any) ([]byte, error) { return schema, nil },
	}

	_, err := s.run(map[string]any{})
	require.Error(t, err, "missing required field should fail validation")
}

func TestSchemaStep_Run_SelectorReceivesValues(t *testing.T) {
	t.Parallel()

	schema := []byte(`{"type": "object"}`)
	var received map[string]any

	s := &schemaStep{
		selector: func(values map[string]any) ([]byte, error) {
			received = values
			return schema, nil
		},
	}

	input := map[string]any{"key": "value"}
	_, err := s.run(input)
	require.NoError(t, err)
	assert.Equal(t, input, received)
}

func TestTransformStep_Kind(t *testing.T) {
	t.Parallel()
	s := &transformStep{}
	assert.Equal(t, "transform", s.kind())
}

func TestTransformStep_Run_Success(t *testing.T) {
	t.Parallel()

	s := &transformStep{
		fn: func(values map[string]any) (map[string]any, error) {
			values["added"] = true
			return values, nil
		},
	}

	got, err := s.run(map[string]any{"existing": "yes"})
	require.NoError(t, err)
	assert.Equal(t, true, got["added"])
	assert.Equal(t, "yes", got["existing"])
}

func TestTransformStep_Run_Error(t *testing.T) {
	t.Parallel()

	fnErr := errors.New("transform boom")
	s := &transformStep{
		fn: func(_ map[string]any) (map[string]any, error) { return nil, fnErr },
	}

	_, err := s.run(map[string]any{})
	require.ErrorIs(t, err, fnErr)
}

func TestTransformStep_Run_ReplacesMap(t *testing.T) {
	t.Parallel()

	replacement := map[string]any{"brand_new": "map"}
	s := &transformStep{
		fn: func(_ map[string]any) (map[string]any, error) { return replacement, nil },
	}

	got, err := s.run(map[string]any{"old": "data"})
	require.NoError(t, err)
	assert.Equal(t, replacement, got)
}

func TestValidatorStep_Kind(t *testing.T) {
	t.Parallel()
	s := &validatorStep{}
	assert.Equal(t, "validator", s.kind())
}

func TestValidatorStep_Run_Success(t *testing.T) {
	t.Parallel()

	s := &validatorStep{
		fn: func(_ map[string]any) error { return nil },
	}

	input := map[string]any{"key": "value"}
	got, err := s.run(input)
	require.NoError(t, err)
	assert.Equal(t, input, got, "values should pass through unmodified")
}

func TestValidatorStep_Run_Error(t *testing.T) {
	t.Parallel()

	fnErr := errors.New("validation failed")
	s := &validatorStep{
		fn: func(_ map[string]any) error { return fnErr },
	}

	got, err := s.run(map[string]any{})
	require.ErrorIs(t, err, fnErr)
	assert.Nil(t, got)
}

func TestValidatorStep_Run_PanicError(t *testing.T) {
	t.Parallel()

	panicErr := errors.New("panic error value")
	s := &validatorStep{
		fn: func(_ map[string]any) error { panic(panicErr) },
	}

	got, err := s.run(map[string]any{})
	require.Error(t, err)
	assert.ErrorIs(t, err, panicErr)
	assert.Contains(t, err.Error(), "validator panic")
	assert.Nil(t, got)
}

func TestValidatorStep_Run_PanicString(t *testing.T) {
	t.Parallel()

	s := &validatorStep{
		fn: func(_ map[string]any) error { panic("oops") },
	}

	got, err := s.run(map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validator panic")
	assert.Contains(t, err.Error(), "oops")
	assert.Nil(t, got)
}

func TestValidatorStep_Run_PanicInt(t *testing.T) {
	t.Parallel()

	s := &validatorStep{
		fn: func(_ map[string]any) error { panic(42) },
	}

	got, err := s.run(map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validator panic")
	assert.Contains(t, err.Error(), "42")
	assert.Nil(t, got)
}

func TestAllStepsImplementStepInterface(t *testing.T) {
	t.Parallel()

	var _ step = (*schemaStep)(nil)
	var _ step = (*transformStep)(nil)
	var _ step = (*validatorStep)(nil)
}
