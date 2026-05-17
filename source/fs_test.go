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

package source

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra/codec"
)

func TestFileFS_Load_YAML(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"app.yaml": &fstest.MapFile{Data: []byte("port: 4242\n")},
	}
	src := NewFileFS(fsys, "app.yaml", codec.YAML)
	m, err := src.Load(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 4242, m["port"])
}

func TestFileFS_Load_nilFS(t *testing.T) {
	t.Parallel()

	src := NewFileFS(nil, "x.yaml", codec.YAML)
	_, err := src.Load(context.Background())
	require.Error(t, err)
}

func TestFileFS_Load_missingFile(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{}
	src := NewFileFS(fsys, "missing.yaml", codec.YAML)
	_, err := src.Load(context.Background())
	require.Error(t, err)
}
