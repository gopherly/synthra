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

	"github.com/stretchr/testify/require"
	"gopherly.dev/synthra"
)

func TestEnvironment_OnlyEnvSource(t *testing.T) {
	t.Setenv("WEBAPP_SERVER_HOST", "localhost")
	t.Setenv("WEBAPP_SERVER_PORT", "8080")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_HOST", "db.example.com")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_PORT", "5432")
	t.Setenv("WEBAPP_DATABASE_PRIMARY_DATABASE", "myapp")
	t.Setenv("WEBAPP_AUTH_JWT_SECRET", "secret")
	t.Setenv("WEBAPP_FEATURES_DEBUG_MODE", "true")

	var sc SimpleConfig
	cfg := synthra.MustNew(
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&sc),
	)
	require.NoError(t, cfg.Load(context.Background()))

	require.Equal(t, "localhost", sc.Server.Host)
	require.Equal(t, 8080, sc.Server.Port)
	require.Equal(t, "db.example.com", sc.Database.Primary.Host)
	require.True(t, sc.Features.Debug.Mode)
}
