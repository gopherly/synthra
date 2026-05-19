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

// Package main demonstrates layered configuration (YAML defaults plus
// environment overrides), struct binding, and validation with Synthra.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gopherly.dev/synthra"
)

// WebAppConfig is the bound configuration struct.
type WebAppConfig struct {
	Server   ServerConfig   `synthra:"server"`
	Database DatabaseConfig `synthra:"database"`
	Auth     AuthConfig     `synthra:"auth"`
	Features FeaturesConfig `synthra:"features"`
}

// ServerConfig holds server network and TLS settings.
type ServerConfig struct {
	Host string    `synthra:"host"`
	Port int       `synthra:"port"`
	TLS  TLSConfig `synthra:"tls"`
}

// TLSConfig holds TLS on/off and certificate paths.
type TLSConfig struct {
	Enabled bool       `synthra:"enabled"`
	Cert    CertConfig `synthra:"cert"`
	Key     KeyConfig  `synthra:"key"`
}

// CertConfig is the path to the TLS certificate file.
type CertConfig struct {
	File string `synthra:"file"`
}

// KeyConfig is the path to the TLS private key file.
type KeyConfig struct {
	File string `synthra:"file"`
}

// DatabaseConfig holds primary connection and pool settings.
type DatabaseConfig struct {
	Primary PrimaryConfig `synthra:"primary"`
	Pool    PoolConfig    `synthra:"pool"`
}

// PrimaryConfig holds primary database connection details.
type PrimaryConfig struct {
	Host     string `synthra:"host"`
	Port     int    `synthra:"port"`
	Database string `synthra:"database"`
}

// PoolConfig holds connection pool limits.
type PoolConfig struct {
	Max MaxConfig `synthra:"max"`
}

// MaxConfig holds maximum open and idle connection counts.
type MaxConfig struct {
	Open int `synthra:"open"`
	Idle int `synthra:"idle"`
}

// AuthConfig holds JWT settings.
type AuthConfig struct {
	JWT   JWTConfig   `synthra:"jwt"`
	Token TokenConfig `synthra:"token"`
}

// JWTConfig holds the JWT signing secret.
type JWTConfig struct {
	Secret string `synthra:"secret"`
}

// TokenConfig holds the access-token lifetime.
type TokenConfig struct {
	Duration time.Duration `synthra:"duration"`
}

// FeaturesConfig holds feature flags.
type FeaturesConfig struct {
	Debug DebugConfig `synthra:"debug"`
}

// DebugConfig controls debug mode.
type DebugConfig struct {
	Mode bool `synthra:"mode"`
}

// Validate implements [synthra.Validator] for startup checks on the bound struct.
func (c *WebAppConfig) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Auth.JWT.Secret == "" {
		return errors.New("auth.jwt.secret is required")
	}
	if c.Server.TLS.Enabled {
		if c.Server.TLS.Cert.File == "" {
			return errors.New("server.tls.cert.file is required when TLS is enabled")
		}
		if c.Server.TLS.Key.File == "" {
			return errors.New("server.tls.key.file is required when TLS is enabled")
		}
	}
	return nil
}

func main() {
	var wc WebAppConfig

	cfg := synthra.MustNew(
		synthra.WithFile("config.yaml"),
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&wc),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("server=%s:%d tls=%v\n", wc.Server.Host, wc.Server.Port, wc.Server.TLS.Enabled)
	fmt.Printf("database=%s:%d/%s\n", wc.Database.Primary.Host, wc.Database.Primary.Port, wc.Database.Primary.Database)
	fmt.Printf("auth.jwt.secret=%s\n", wc.Auth.JWT.Secret)
	fmt.Printf("debug=%v\n", wc.Features.Debug.Mode)
}
