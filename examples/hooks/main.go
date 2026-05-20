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

// Package main demonstrates all three hook points in one pipeline.
//
// Pipeline order:
//
//  1. WithTransform  — map stage: override log level for production.
//  2. WithValidator  — cross-field: TLS cert + key must both be set
//     when TLS is enabled.
//  3. WithBinding    — decode map into App struct.
//  4. OnBound[App]   — struct stage: normalize Logging.Level to lowercase.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"gopherly.dev/synthra"
)

// App is the bound configuration struct.
type App struct {
	Env     string        `synthra:"env"`
	Logging LoggingConfig `synthra:"logging"`
	Server  ServerConfig  `synthra:"server"`
}

// LoggingConfig holds the log level.
type LoggingConfig struct {
	Level string `synthra:"level"`
}

// ServerConfig holds TLS settings.
type ServerConfig struct {
	TLS TLSConfig `synthra:"tls"`
}

// TLSConfig holds TLS enabled flag and certificate paths.
type TLSConfig struct {
	Enabled bool       `synthra:"enabled"`
	Cert    CertConfig `synthra:"cert"`
	Key     KeyConfig  `synthra:"key"`
}

// CertConfig holds the TLS certificate file path.
type CertConfig struct {
	File string `synthra:"file"`
}

// KeyConfig holds the TLS private key file path.
type KeyConfig struct {
	File string `synthra:"file"`
}

func main() {
	var app App

	cfg := synthra.MustNew(
		synthra.WithFile("config.yaml"),

		// Map stage: force log level to "warn" in production before binding.
		synthra.WithTransform(func(_ context.Context, v *synthra.Configurable) error {
			if v.StringOr("env", "dev") == "prod" {
				return v.Set("logging.level", "warn")
			}
			return nil
		}),

		// Cross-field validation: TLS cert and key must both be present when TLS is on.
		synthra.WithValidator(func(_ context.Context, c *synthra.Configuration) error {
			enabled := strings.EqualFold(fmt.Sprint(c.Get("server.tls.enabled")), "true")
			if !enabled {
				return nil
			}
			cert := strings.TrimSpace(c.StringOr("server.tls.cert.file", ""))
			key := strings.TrimSpace(c.StringOr("server.tls.key.file", ""))
			if cert == "" || key == "" {
				return errors.New("server.tls.cert.file and server.tls.key.file are required when TLS is enabled")
			}
			return nil
		}),

		// Binding stage: decode map into App; normalize level to lowercase afterwards.
		synthra.WithBinding(&app,
			synthra.OnBound(func(a *App) error {
				a.Logging.Level = strings.ToLower(a.Logging.Level)
				return nil
			}),
		),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}

	fmt.Printf("env=%s  logging.level=%s  tls=%v\n",
		app.Env, app.Logging.Level, app.Server.TLS.Enabled)
}
