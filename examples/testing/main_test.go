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
	"testing"

	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
	"gopherly.dev/synthra/source"
	"gopherly.dev/synthra/synthratest"
)

func TestSynthraTestConfig_LoadsMockSource(t *testing.T) {
	cfg := synthratest.Config(t,
		synthra.WithSource(source.NewMap(map[string]any{
			"server": map[string]any{"port": 8080, "host": "127.0.0.1"},
		})),
	)
	require.NoError(t, cfg.Load(t.Context()))
	port, err := cfg.Int("server.port")
	require.NoError(t, err)
	require.Equal(t, 8080, port)
	host, err := cfg.String("server.host")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", host)
}
