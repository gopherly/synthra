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

	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
)

func TestConsulWithIf_FileAndEnvWithoutConsul(t *testing.T) {
	t.Setenv("EDGE_SERVICE_PORT", "9090")

	cfg := synthra.MustNew(
		synthra.WithFile("config.yaml"),
		synthra.WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "", synthra.WithConsul("synthra/example/config.yaml")),
		synthra.WithEnv("EDGE_"),
	)
	require.NoError(t, cfg.Load(context.Background()))
	svcName, err := cfg.String("service.name")
	require.NoError(t, err)
	require.Equal(t, "edge", svcName)
	svcPort, err := cfg.Int("service.port")
	require.NoError(t, err)
	require.Equal(t, 9090, svcPort)
}
