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

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
)

func TestBasic_YAMLBinding(t *testing.T) {
	var cfg Config
	c := synthra.MustNew(
		synthra.WithFile("config.yaml"),
		synthra.WithBinding(&cfg),
	)
	require.NoError(t, c.Load(context.Background()))

	require.Equal(t, "bar", cfg.Foo)
	require.Equal(t, 10*time.Second, cfg.Timeout)
	require.True(t, cfg.Debug)
	require.Equal(t, []string{"admin", "user"}, cfg.Roles)
	require.Equal(t, []string{"x1", "x2", "x3"}, cfg.Types)
	require.Equal(t, "http://localhost:8080", cfg.Worker.Address.String())
	// worker.timeout is a bare integer in YAML; decoding follows Synthra scalar rules.
	require.NotZero(t, cfg.Worker.Timeout)
}
