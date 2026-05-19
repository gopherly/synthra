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

//go:build !integration

package synthra_test

import (
	"context"
	"fmt"
	"log"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/codec"
	"gopherly.dev/synthra/synthratest"
)

func exampleString(cfg *synthra.Synthra, key string) string {
	v, err := cfg.String(key)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func exampleInt(cfg *synthra.Synthra, key string) int {
	v, err := cfg.Int(key)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func exampleBool(cfg *synthra.Synthra, key string) bool {
	v, err := cfg.Bool(key)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func exampleStringSlice(cfg *synthra.Synthra, key string) []string {
	v, err := cfg.StringSlice(key)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func exampleStringMap(cfg *synthra.Synthra, key string) map[string]any {
	v, err := cfg.StringMap(key)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

// Example demonstrates basic configuration usage.
func Example() {
	// Create config with YAML content
	yamlContent := []byte(`
server:
  host: localhost
  port: 8080
database:
  name: mydb
`)

	cfg, err := synthra.New(
		synthra.WithContent(yamlContent, codec.YAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Load configuration
	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Access values
	fmt.Println(exampleString(cfg, "server.host"))
	fmt.Println(exampleInt(cfg, "server.port"))
	fmt.Println(exampleString(cfg, "database.name"))

	// Output:
	// localhost
	// 8080
	// mydb
}

// ExampleNew demonstrates creating a new configuration instance.
func ExampleNew() {
	cfg, err := synthra.New()
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Synthra created successfully")
	// Output: Synthra created successfully
}

// ExampleMustNew demonstrates creating a configuration instance with
// panic on error.
func ExampleMustNew() {
	cfg := synthra.MustNew()
	if err := cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Synthra created successfully")
	// Output: Synthra created successfully
}

// ExampleSynthra_Load demonstrates loading configuration.
func ExampleSynthra_Load() {
	cfg := synthra.MustNew(
		synthra.WithContent([]byte(`{"app": "example", "port": 8080}`), codec.JSON),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	app := exampleString(cfg, "app")
	port := exampleInt(cfg, "port")
	fmt.Printf("App: %s, Port: %d\n", app, port)
	// Output: App: example, Port: 8080
}

// ExampleSynthra_Dump demonstrates writing configuration to registered dumpers.
func ExampleSynthra_Dump() {
	// Create a mock dumper for demonstration
	dumper := &synthratest.Dumper{}

	cfg := synthra.MustNew(
		synthra.WithContent([]byte(`{"service": "api", "version": "1.0"}`), codec.JSON),
		synthra.WithDumper(dumper),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	if err := cfg.Dump(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Configuration dumped successfully")
	// Output: Configuration dumped successfully
}

// ExampleWithFile demonstrates loading configuration from a file.
func ExampleWithFile() {
	// Create a temporary config file (in real code, use an actual file path)
	cfg, err := synthra.New(
		synthra.WithContent([]byte(`{"name": "example"}`), codec.JSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println(exampleString(cfg, "name"))
	// Output: example
}

// ExampleWithContent demonstrates loading configuration from byte content.
func ExampleWithContent() {
	jsonContent := []byte(`{
		"app": {
			"name": "MyApp",
			"version": "1.0.0"
		}
	}`)

	cfg, err := synthra.New(
		synthra.WithContent(jsonContent, codec.JSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println(exampleString(cfg, "app.name"))
	fmt.Println(exampleString(cfg, "app.version"))
	// Output:
	// MyApp
	// 1.0.0
}

// ExampleWithBinding demonstrates binding configuration to a struct.
func ExampleWithBinding() {
	type ServerConfig struct {
		Host string `synthra:"host"`
		Port int    `synthra:"port"`
	}

	type AppConfig struct {
		Server ServerConfig `synthra:"server"`
	}

	yamlContent := []byte(`
server:
  host: localhost
  port: 8080
`)

	var appConfig AppConfig
	cfg, err := synthra.New(
		synthra.WithContent(yamlContent, codec.YAML),
		synthra.WithBinding(&appConfig),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s:%d\n", appConfig.Server.Host, appConfig.Server.Port)
	// Output: localhost:8080
}

// ExampleWithValidator demonstrates using a custom validator.
func ExampleWithValidator() {
	yamlContent := []byte(`name: myapp`)

	cfg, err := synthra.New(
		synthra.WithContent(yamlContent, codec.YAML),
		synthra.WithValidator(func(v *synthra.Values) error {
			// Custom validation logic
			if !v.Has("name") {
				return fmt.Errorf("name is required")
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Validation passed")
	// Output: Validation passed
}

// ExampleWithJSONSchema validates the merged configuration against a JSON
// Schema. Load fails fast if required keys are missing or values have the
// wrong type.
func ExampleWithJSONSchema() {
	schema := []byte(`{
		"type": "object",
		"required": ["service", "port"],
		"properties": {
			"service": {"type": "string", "minLength": 1},
			"port":    {"type": "integer", "minimum": 1, "maximum": 65535}
		}
	}`)

	cfg := synthra.MustNew(
		synthra.WithContent([]byte("service: api\nport: 8080\n"), codec.YAML),
		synthra.WithJSONSchema(schema),
	)
	if err := cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("service=%s port=%d\n", exampleString(cfg, "service"), exampleInt(cfg, "port"))
	// Output: service=api port=8080
}

// ExampleSynthra_Get demonstrates retrieving configuration values.
func ExampleSynthra_Get() {
	yamlContent := []byte(`
settings:
  enabled: true
  count: 42
`)

	cfg, err := synthra.New(
		synthra.WithContent(yamlContent, codec.YAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg.Get("settings.enabled"))
	fmt.Println(cfg.Get("settings.count"))
	// Output:
	// true
	// 42
}

// ExampleSynthra_String demonstrates retrieving string values.
func ExampleSynthra_String() {
	jsonContent := []byte(`{"name": "MyApp", "env": "production"}`)

	cfg, err := synthra.New(
		synthra.WithContent(jsonContent, codec.JSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println(exampleString(cfg, "name"))
	fmt.Println(exampleString(cfg, "env"))
	// Output:
	// MyApp
	// production
}

// ExampleSynthra_Int demonstrates retrieving integer values.
func ExampleSynthra_Int() {
	jsonContent := []byte(`{"port": 8080, "workers": 4}`)

	cfg, err := synthra.New(
		synthra.WithContent(jsonContent, codec.JSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println(exampleInt(cfg, "port"))
	fmt.Println(exampleInt(cfg, "workers"))
	// Output:
	// 8080
	// 4
}

// ExampleSynthra_Bool demonstrates retrieving boolean values.
func ExampleSynthra_Bool() {
	jsonContent := []byte(`{"debug": true, "verbose": false}`)

	cfg, err := synthra.New(
		synthra.WithContent(jsonContent, codec.JSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println(exampleBool(cfg, "debug"))
	fmt.Println(exampleBool(cfg, "verbose"))
	// Output:
	// true
	// false
}

// ExampleSynthra_StringSlice demonstrates retrieving string slices.
func ExampleSynthra_StringSlice() {
	yamlContent := []byte(`
tags:
  - web
  - api
  - backend
`)

	cfg, err := synthra.New(
		synthra.WithContent(yamlContent, codec.YAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	tags := exampleStringSlice(cfg, "tags")
	fmt.Printf("%v\n", tags)
	// Output: [web api backend]
}

// ExampleSynthra_StringMap demonstrates retrieving string maps.
func ExampleSynthra_StringMap() {
	yamlContent := []byte(`
metadata:
  author: John Doe
  version: 1.0.0
`)

	cfg, err := synthra.New(
		synthra.WithContent(yamlContent, codec.YAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	metadata := exampleStringMap(cfg, "metadata")
	fmt.Println(metadata["author"])
	fmt.Println(metadata["version"])
	// Output:
	// John Doe
	// 1.0.0
}

// Example_multipleSources demonstrates merging multiple configuration sources.
func Example_multipleSources() {
	// Base configuration
	baseConfig := []byte(`
server:
  host: localhost
  port: 8080
`)

	// Override configuration
	overrideConfig := []byte(`
server:
  port: 9090
`)

	cfg, err := synthra.New(
		synthra.WithContent(baseConfig, codec.YAML),
		synthra.WithContent(overrideConfig, codec.YAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Later sources override earlier ones
	fmt.Println(exampleString(cfg, "server.host"))
	fmt.Println(exampleInt(cfg, "server.port"))
	// Output:
	// localhost
	// 9090
}

// Example_environmentVariables demonstrates loading configuration from
// environment variables.
func Example_environmentVariables() {
	// In real usage, set environment variables like:
	// export APP_SERVER_HOST=localhost
	// export APP_SERVER_PORT=8080

	cfg, err := synthra.New(
		synthra.WithEnv("APP_"),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Access environment variables without the prefix
	// e.g., APP_SERVER_HOST becomes server.host
	fmt.Println("Environment variables loaded")
	// Output: Environment variables loaded
}
