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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/codec"
)

func buildCfg(t *testing.T, yaml []byte) (*synthra.Synthra, *App) {
	t.Helper()
	var app App
	cfg := synthra.MustNew(
		synthra.WithContent(yaml, codec.YAML),
		synthra.WithTransform(func(v *synthra.Values) error {
			if v.StringOr("env", "dev") == "prod" {
				return v.Set("logging.level", "warn")
			}
			return nil
		}),
		synthra.WithValidator(func(r synthra.Reader) error {
			enabled := strings.EqualFold(fmt.Sprint(r.Get("server.tls.enabled")), "true")
			if !enabled {
				return nil
			}
			cert := strings.TrimSpace(r.StringOr("server.tls.cert.file", ""))
			key := strings.TrimSpace(r.StringOr("server.tls.key.file", ""))
			if cert == "" || key == "" {
				return errors.New("server.tls.cert.file and server.tls.key.file are required when TLS is enabled")
			}
			return nil
		}),
		synthra.WithBinding(&app,
			synthra.OnBound(func(a *App) error {
				a.Logging.Level = strings.ToLower(a.Logging.Level)
				return nil
			}),
		),
	)
	return cfg, &app
}

func TestHooks_HappyPath(t *testing.T) {
	yaml := []byte(`
env: dev
logging:
  level: INFO
server:
  tls:
    enabled: true
    cert:
      file: "/etc/ssl/certs/server.crt"
    key:
      file: "/etc/ssl/private/server.key"
`)
	cfg, app := buildCfg(t, yaml)
	require.NoError(t, cfg.Load(context.Background()))

	assert.Equal(t, "dev", app.Env)
	assert.Equal(t, "info", app.Logging.Level, "OnBound should lowercase the level")
	assert.True(t, app.Server.TLS.Enabled)
}

func TestHooks_WithTransform_ProdOverridesLevel(t *testing.T) {
	yaml := []byte(`
env: prod
logging:
  level: DEBUG
server:
  tls:
    enabled: false
`)
	cfg, app := buildCfg(t, yaml)
	require.NoError(t, cfg.Load(context.Background()))

	assert.Equal(t, "warn", app.Logging.Level, "WithTransform should override level to warn in prod")
}

func TestHooks_WithValidator_TLSMissingCert(t *testing.T) {
	yaml := []byte(`
env: dev
logging:
  level: info
server:
  tls:
    enabled: true
    cert:
      file: ""
    key:
      file: ""
`)
	cfg, _ := buildCfg(t, yaml)
	err := cfg.Load(context.Background())
	require.Error(t, err, "validator should reject TLS enabled without cert/key")
}

func TestHooks_OnBound_NormalizesLevel(t *testing.T) {
	yaml := []byte(`
env: dev
logging:
  level: WARN
server:
  tls:
    enabled: false
`)
	cfg, app := buildCfg(t, yaml)
	require.NoError(t, cfg.Load(context.Background()))

	assert.Equal(t, "warn", app.Logging.Level, "OnBound should lowercase WARN")
}
