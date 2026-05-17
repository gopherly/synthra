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

// WebAppConfig represents a complete web application configuration
// that can be populated from environment variables
type WebAppConfig struct {
	Server     ServerConfig     `synthra:"server"`
	Database   DatabaseConfig   `synthra:"database"`
	Redis      RedisConfig      `synthra:"redis"`
	Auth       AuthConfig       `synthra:"auth"`
	Logging    LoggingConfig    `synthra:"logging"`
	Monitoring MonitoringConfig `synthra:"monitoring"`
	Features   FeaturesConfig   `synthra:"features"`
}

// ServerConfig represents server configuration settings
type ServerConfig struct {
	Host         string        `synthra:"host"`
	Port         int           `synthra:"port"`
	ReadTimeout  time.Duration `synthra:"read.timeout"`
	WriteTimeout time.Duration `synthra:"write.timeout"`
	TLS          TLSConfig     `synthra:"tls"`
}

// TLSConfig represents TLS/SSL configuration settings
type TLSConfig struct {
	Enabled bool       `synthra:"enabled"`
	Cert    CertConfig `synthra:"cert"`
	Key     KeyConfig  `synthra:"key"`
}

// CertConfig represents TLS certificate configuration
type CertConfig struct {
	File string `synthra:"file"`
}

// KeyConfig represents TLS private key configuration
type KeyConfig struct {
	File string `synthra:"file"`
}

// DatabaseConfig represents database configuration settings
type DatabaseConfig struct {
	Primary PrimaryConfig `synthra:"primary"`
	Replica ReplicaConfig `synthra:"replica"`
	Pool    PoolConfig    `synthra:"pool"`
}

// PrimaryConfig represents primary database connection settings
type PrimaryConfig struct {
	Host     string `synthra:"host"`
	Port     int    `synthra:"port"`
	Database string `synthra:"database"`
	Username string `synthra:"username"`
	Password string `synthra:"password"`
	SSLMode  string `synthra:"ssl.mode"`
}

// ReplicaConfig represents replica database connection settings
type ReplicaConfig struct {
	Host     string `synthra:"host"`
	Port     int    `synthra:"port"`
	Database string `synthra:"database"`
	Username string `synthra:"username"`
	Password string `synthra:"password"`
	SSLMode  string `synthra:"ssl.mode"`
}

// PoolConfig represents database connection pool settings
type PoolConfig struct {
	Max MaxConfig `synthra:"max"`
}

// MaxConfig represents maximum connection pool limits
type MaxConfig struct {
	Open     int           `synthra:"open"`
	Idle     int           `synthra:"idle"`
	Lifetime time.Duration `synthra:"lifetime"`
}

// RedisConfig represents Redis connection settings
type RedisConfig struct {
	Host     string        `synthra:"host"`
	Port     int           `synthra:"port"`
	Password string        `synthra:"password"`
	Database int           `synthra:"database"`
	Timeout  time.Duration `synthra:"timeout"`
}

// AuthConfig represents authentication configuration settings
type AuthConfig struct {
	JWT     JWTConfig     `synthra:"jwt"`
	Token   TokenConfig   `synthra:"token"`
	Refresh RefreshConfig `synthra:"refresh"`
}

// JWTConfig represents JWT authentication settings
type JWTConfig struct {
	Secret string `synthra:"secret"`
}

// TokenConfig represents token configuration settings
type TokenConfig struct {
	Duration time.Duration `synthra:"duration"`
}

// RefreshConfig represents refresh token configuration settings
type RefreshConfig struct {
	Secret string `synthra:"secret"`
}

// LoggingConfig represents logging configuration settings
type LoggingConfig struct {
	Level      string `synthra:"level"`
	Format     string `synthra:"format"`
	OutputFile string `synthra:"output.file"`
}

// MonitoringConfig represents monitoring and metrics configuration
type MonitoringConfig struct {
	Enabled     bool   `synthra:"enabled"`
	MetricsPort int    `synthra:"metrics.port"`
	HealthPath  string `synthra:"health.path"`
}

// FeaturesConfig represents feature flags and settings
type FeaturesConfig struct {
	RateLimit RateLimitConfig `synthra:"rate.limit"`
	Cache     CacheConfig     `synthra:"cache"`
	Debug     DebugConfig     `synthra:"debug"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled bool `synthra:"enabled"`
}

// CacheConfig represents caching configuration
type CacheConfig struct {
	Enabled bool `synthra:"enabled"`
}

// DebugConfig represents debug mode settings
type DebugConfig struct {
	Mode bool `synthra:"mode"`
}

// PrintConfig displays the configuration in a readable format
func (c *WebAppConfig) PrintConfig() {
	fmt.Println("=== Web Application Configuration (YAML + Environment Variables) ===")
	fmt.Printf("Server: %s:%d\n", c.Server.Host, c.Server.Port)
	fmt.Printf("  Read Timeout: %v\n", c.Server.ReadTimeout)
	fmt.Printf("  Write Timeout: %v\n", c.Server.WriteTimeout)
	fmt.Printf("  TLS Enabled: %t\n", c.Server.TLS.Enabled)
	if c.Server.TLS.Enabled {
		fmt.Printf("  TLS Cert: %s\n", c.Server.TLS.Cert.File)
		fmt.Printf("  TLS Key: %s\n", c.Server.TLS.Key.File)
	}

	fmt.Printf("\nDatabase Primary: %s:%d/%s\n",
		c.Database.Primary.Host, c.Database.Primary.Port, c.Database.Primary.Database)
	fmt.Printf("Database Replica: %s:%d/%s\n",
		c.Database.Replica.Host, c.Database.Replica.Port, c.Database.Replica.Database)
	fmt.Printf("Database Pool: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v\n",
		c.Database.Pool.Max.Open, c.Database.Pool.Max.Idle, c.Database.Pool.Max.Lifetime)

	fmt.Printf("\nRedis: %s:%d (DB: %d)\n", c.Redis.Host, c.Redis.Port, c.Redis.Database)
	fmt.Printf("Redis Timeout: %v\n", c.Redis.Timeout)

	fmt.Printf("\nAuth Token Duration: %v\n", c.Auth.Token.Duration)
	fmt.Printf("Logging Level: %s, Format: %s\n", c.Logging.Level, c.Logging.Format)
	if c.Logging.OutputFile != "" {
		fmt.Printf("Logging Output: %s\n", c.Logging.OutputFile)
	}

	fmt.Printf("\nMonitoring Enabled: %t\n", c.Monitoring.Enabled)
	if c.Monitoring.Enabled {
		fmt.Printf("Metrics Port: %d\n", c.Monitoring.MetricsPort)
		fmt.Printf("Health Path: %s\n", c.Monitoring.HealthPath)
	}

	fmt.Printf("\nFeatures:\n")
	fmt.Printf("  Rate Limit: %t\n", c.Features.RateLimit.Enabled)
	fmt.Printf("  Cache: %t\n", c.Features.Cache.Enabled)
	fmt.Printf("  Debug Mode: %t\n", c.Features.Debug.Mode)
	fmt.Println("=====================================")
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

	// Create configuration with multiple sources
	cfg := synthra.MustNew(
		// First, load from YAML file (default values)
		synthra.WithFile("config.yaml"),
		// Then, override with environment variables (higher precedence)
		synthra.WithEnv("WEBAPP_"),
		// Bind to our struct
		synthra.WithBinding(&wc),
	)

	// Load configuration
	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print the loaded configuration
	wc.PrintConfig()

	// Demonstrate accessing configuration values directly
	fmt.Println("\n=== Direct Configuration Access ===")
	serverHost, err := cfg.String("server.host")
	if err != nil {
		log.Fatalf("server.host: %v", err)
	}
	serverPort, err := cfg.Int("server.port")
	if err != nil {
		log.Fatalf("server.port: %v", err)
	}
	databaseHost, err := cfg.String("database.primary.host")
	if err != nil {
		log.Fatalf("database.primary.host: %v", err)
	}

	fmt.Printf("Server: %s:%d\n", serverHost, serverPort)
	fmt.Printf("Database: %s\n", databaseHost)

	// Check if TLS is enabled
	tlsEnabled, err := cfg.Bool("server.tls.enabled")
	if err != nil {
		log.Fatalf("server.tls.enabled: %v", err)
	}
	if tlsEnabled {
		fmt.Println("TLS is enabled")
	} else {
		fmt.Println("TLS is disabled")
	}

	// Demonstrate configuration precedence
	fmt.Println("\n=== Configuration Precedence Demo ===")
	fmt.Println("Values are loaded in this order:")
	fmt.Println("1. YAML file (config.yaml) - default values")
	fmt.Println("2. Environment variables (WEBAPP_*) - override defaults")
	fmt.Println("")
	fmt.Println("Example: If YAML has server.port=3000 and env has WEBAPP_SERVER_PORT=8080")
	fmt.Println("The final value will be 8080 (environment variable wins)")
	fmt.Println("")
	fmt.Println("Env keys: strip the prefix, split on underscores, nest (e.g.")
	fmt.Println("WEBAPP_DATABASE_PRIMARY_HOST -> database.primary.host).")
}
