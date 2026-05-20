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
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
)

// buildCfg creates a Synthra instance using the two-phase validation pipeline
// with the provided YAML content as the manifest source.
func buildCfg(t *testing.T, yamlContent []byte, vars map[string]string) (*synthra.Synthra, error) {
	t.Helper()
	fsys := fstest.MapFS{
		"manifest.yaml": &fstest.MapFile{Data: yamlContent},
	}
	return synthra.New(
		synthra.WithFileFS(fsys, "manifest.yaml"),
		synthra.WithJSONSchemaFunc(func(_ context.Context, _ *synthra.Configurable) ([]byte, error) {
			return environmentsSchema, nil
		}),
		synthra.WithEnvSubst(synthra.FromMap(vars)),
		synthra.WithJSONSchemaFunc(func(_ context.Context, _ *synthra.Configurable) ([]byte, error) {
			return manifestSchema, nil
		}),
	)
}

func TestMultiSchema_HappyPath(t *testing.T) {
	t.Parallel()

	yaml := []byte(`
apiversion: v1-alpha.1
service: my-service
port: 9090
environments:
  - name: production
    envFile: .env.production
  - name: staging
`)
	cfg, err := buildCfg(t, yaml, nil)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	apiVersion, err := cfg.String("apiversion")
	require.NoError(t, err)
	assert.Equal(t, "v1-alpha.1", apiVersion)

	service, err := cfg.String("service")
	require.NoError(t, err)
	assert.Equal(t, "my-service", service)

	port, err := cfg.Int("port")
	require.NoError(t, err)
	assert.Equal(t, 9090, port)
}

func TestMultiSchema_PlaceholderSubstituted(t *testing.T) {
	t.Parallel()

	yaml := []byte(`
apiversion: v1-alpha.1
service: ${MY_SVC}
environments:
  - name: production
`)
	cfg, err := buildCfg(t, yaml, map[string]string{"MY_SVC": "resolved-service"})
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	service, err := cfg.String("service")
	require.NoError(t, err)
	assert.Equal(t, "resolved-service", service)
}

func TestMultiSchema_MissingEnvironmentsFailsFirstSchema(t *testing.T) {
	t.Parallel()

	// Missing "environments" — should fail at step[0]:schema (pre-substitution check).
	yaml := []byte(`
apiversion: v1-alpha.1
service: my-service
`)
	cfg, err := buildCfg(t, yaml, nil)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *synthra.ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, "step[0]:schema", ce.Path)
}

func TestMultiSchema_MissingServiceFailsSecondSchema(t *testing.T) {
	t.Parallel()

	// "environments" present (passes first schema), but "service" is absent
	// (fails second schema after substitution).
	yaml := []byte(`
apiversion: v1-alpha.1
environments:
  - name: production
`)
	cfg, err := buildCfg(t, yaml, nil)
	require.NoError(t, err)

	loadErr := cfg.Load(context.Background())
	require.Error(t, loadErr)

	var ce *synthra.ConfigError
	require.ErrorAs(t, loadErr, &ce)
	assert.Equal(t, "step[2]:schema", ce.Path)
}

func TestMultiSchema_DefaultAppliedByFirstSchema(t *testing.T) {
	t.Parallel()

	// "envFile" default (.env) should be applied by the first schema step.
	yaml := []byte(`
apiversion: v1-alpha.1
service: my-service
environments:
  - name: production
`)
	cfg, err := buildCfg(t, yaml, nil)
	require.NoError(t, err)
	require.NoError(t, cfg.Load(context.Background()))

	envs, ok := cfg.Get("environments").([]any)
	require.True(t, ok)
	require.Len(t, envs, 1)

	env, ok := envs[0].(map[string]any)
	require.True(t, ok)
	// envFile key matches the schema's canonical casing.
	assert.Equal(t, ".env", env["envFile"])
}
