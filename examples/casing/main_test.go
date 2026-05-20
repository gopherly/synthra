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

package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/codec"
)

func TestCasing_WithoutSchema_FirstSourceCasingWins(t *testing.T) {
	cfg, err := synthra.New(
		synthra.WithFile("config-base.yaml"),
		synthra.WithFile("config-override.yaml"),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// Case-insensitive access: both casings return the same value.
	v1, err := cfg.String("ApiVersion")
	require.NoError(t, err)
	v2, err := cfg.String("apiVersion")
	require.NoError(t, err)
	assert.Equal(t, v1, v2, "case-insensitive access must return the same value")
	assert.Equal(t, "v2", v1, "override file value must win")
}

func TestCasing_WithSchema_SchemaCasingWins(t *testing.T) {
	schema, err := os.ReadFile("schema.json")
	require.NoError(t, err)

	cfg, err := synthra.New(
		synthra.WithFile("config-base.yaml"),
		synthra.WithFile("config-override.yaml"),
		synthra.WithJSONSchema(schema),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	// Schema declares "apiVersion"; the mixed-case "ApiVersion" key from the
	// base file is canonicalized to "apiVersion" before validation runs.
	apiVer, err := cfg.String("apiVersion")
	require.NoError(t, err)
	assert.Equal(t, "v2", apiVer)

	logLvl, err := cfg.String("logLevel")
	require.NoError(t, err)
	assert.Equal(t, "warn", logLvl)
}

func TestCasing_WithSchema_DefaultApplied(t *testing.T) {
	schema, err := os.ReadFile("schema.json")
	require.NoError(t, err)

	// A config that sets service and apiVersion but omits logLevel; the
	// schema default "info" is applied automatically.
	cfg, err := synthra.New(
		synthra.WithContent([]byte("apiVersion: v1\nservice: svc\n"), codec.YAML),
		synthra.WithJSONSchema(schema),
	)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	logLvl, err := cfg.String("logLevel")
	require.NoError(t, err)
	assert.Equal(t, "info", logLvl, "schema default must be applied when key is absent")
}
