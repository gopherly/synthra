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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/codec"
)

func TestCodecs_MergeJSONThenTOML(t *testing.T) {
	cfg := synthra.MustNew(
		synthra.WithFileAs("app.json", codec.JSON),
		synthra.WithFileAs("overrides.toml", codec.TOML),
	)
	require.NoError(t, cfg.Load(context.Background()))

	app, err := cfg.String("app")
	require.NoError(t, err)
	require.Equal(t, "formats-demo", app)

	// TOML overrides the listen.port from 3000 (JSON) to 4000 (TOML).
	listenPort, err := cfg.Int("listen.port")
	require.NoError(t, err)
	require.Equal(t, 4000, listenPort)

	region, err := cfg.String("meta.region")
	require.NoError(t, err)
	require.Equal(t, "local", region)
}

func TestCodecs_DumpWritesMergedYAML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "effective.yaml")

	cfg := synthra.MustNew(
		synthra.WithFileAs("app.json", codec.JSON),
		synthra.WithFileAs("overrides.toml", codec.TOML),
		synthra.WithFileDumperAs(out, codec.YAML),
	)
	require.NoError(t, cfg.Load(context.Background()))
	require.NoError(t, cfg.Dump(context.Background()))

	raw, err := os.ReadFile(out) // #nosec G304 -- path is t.TempDir() output from this test
	require.NoError(t, err)

	content := strings.ToLower(string(raw))
	require.Contains(t, content, "formats-demo")
	require.Contains(t, content, "4000")
	require.Contains(t, content, "local")
}
