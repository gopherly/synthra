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

func TestDump_WritesMergedYAML(t *testing.T) {
	t.Setenv("APP_SERVER_PORT", "9090")

	out := filepath.Join(t.TempDir(), "out.yaml")
	cfg := synthra.MustNew(
		synthra.WithFile("config.yaml"),
		synthra.WithEnv("APP_"),
		synthra.WithFileDumperAs(out, codec.YAML),
	)
	require.NoError(t, cfg.Load(context.Background()))
	require.NoError(t, cfg.Dump(context.Background()))

	raw, err := os.ReadFile(out) // #nosec G304 -- path is t.TempDir() output from this test
	require.NoError(t, err)
	require.Contains(t, strings.ToLower(string(raw)), "9090")
}
