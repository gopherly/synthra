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

// Package main demonstrates loading configuration from environment
// variables with Synthra.
package main

import (
	"context"
	"fmt"
	"log"

	"gopherly.dev/synthra"
)

// SimpleConfig represents a simple configuration without validation
type SimpleConfig struct {
	Server   ServerConfig   `synthra:"server"`
	Database DatabaseConfig `synthra:"database"`
	Auth     AuthConfig     `synthra:"auth"`
	Features FeaturesConfig `synthra:"features"`
}

// ServerConfig represents server configuration settings
type ServerConfig struct {
	Host string `synthra:"host"`
	Port int    `synthra:"port"`
}

// DatabaseConfig represents database configuration settings
type DatabaseConfig struct {
	Primary PrimaryConfig `synthra:"primary"`
}

// PrimaryConfig represents primary database connection settings
type PrimaryConfig struct {
	Host     string `synthra:"host"`
	Port     int    `synthra:"port"`
	Database string `synthra:"database"`
}

// AuthConfig represents authentication configuration settings
type AuthConfig struct {
	JWT JWTConfig `synthra:"jwt"`
}

// JWTConfig represents JWT authentication settings
type JWTConfig struct {
	Secret string `synthra:"secret"`
}

// FeaturesConfig represents feature flags and settings
type FeaturesConfig struct {
	Debug DebugConfig `synthra:"debug"`
}

// DebugConfig represents debug mode settings
type DebugConfig struct {
	Mode bool `synthra:"mode"`
}

// PrintConfig displays the configuration in a readable format
func (c *SimpleConfig) PrintConfig() {
	fmt.Println("=== Simple Configuration ===")
	fmt.Printf("Server: %s:%d\n", c.Server.Host, c.Server.Port)
	fmt.Printf("Database: %s:%d/%s\n", c.Database.Primary.Host, c.Database.Primary.Port, c.Database.Primary.Database)
	fmt.Printf("Auth JWT Secret: %s\n", c.Auth.JWT.Secret)
	fmt.Printf("Debug Mode: %t\n", c.Features.Debug.Mode)
	fmt.Println("============================")
}

func main() {
	var sc SimpleConfig

	// Create configuration with environment variable source
	cfg := synthra.MustNew(
		synthra.WithEnv("WEBAPP_"),
		synthra.WithBinding(&sc),
	)

	// Load configuration
	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print the loaded configuration
	sc.PrintConfig()

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

	// Check if debug mode is enabled
	debugMode, err := cfg.Bool("features.debug.mode")
	if err != nil {
		log.Fatalf("features.debug.mode: %v", err)
	}
	if debugMode {
		fmt.Println("Debug mode is enabled")
	} else {
		fmt.Println("Debug mode is disabled")
	}
}
