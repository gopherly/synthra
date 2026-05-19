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

// Package main demonstrates two-phase validation using WithJSONSchemaFunc.
//
// The pipeline:
//  1. Load manifest YAML.
//  2. Validate the "environments" block before variable substitution (the raw
//     values must not contain unexpanded placeholders in required fields).
//  3. Substitute ${VAR} placeholders from OS environment variables.
//  4. Validate the fully-substituted manifest against the complete schema.
//
// This pattern is useful when different fields have different substitution
// requirements: some fields must already be concrete before substitution
// (e.g. environment definitions), while others may contain placeholders that
// are expanded in a later step.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"gopherly.dev/synthra"
)

// environmentsSchema validates that the "environments" key is present and is
// an array before variable substitution runs.
var environmentsSchema = []byte(`{
	"type": "object",
	"required": ["apiversion", "environments"],
	"properties": {
		"apiversion":   {"type": "string"},
		"environments": {
			"type": "array",
			"items": {
				"type": "object",
				"required": ["name"],
				"properties": {
					"name":    {"type": "string"},
					"envFile": {"type": "string", "default": ".env"}
				}
			}
		}
	}
}`)

// manifestSchema validates the fully-substituted manifest, including the
// "service" field which may contain ${VAR} placeholders before substitution.
var manifestSchema = []byte(`{
	"type": "object",
	"required": ["apiversion", "service", "environments"],
	"properties": {
		"apiversion":   {"type": "string"},
		"service":      {"type": "string"},
		"port":         {"type": "integer", "default": 8080},
		"environments": {
			"type": "array",
			"items": {
				"type": "object",
				"required": ["name"],
				"properties": {
					"name":    {"type": "string"},
					"envFile": {"type": "string", "default": ".env"}
				}
			}
		}
	}
}`)

func main() {
	path := "manifest.yaml"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	cfg, err := synthra.New(
		synthra.WithFile(path),

		// Step 1 — validate raw environments block before substitution.
		// This schema requires "environments" to be present and structurally
		// valid even before ${VAR} placeholders are expanded.
		synthra.WithJSONSchemaFunc(func(_ *synthra.Values) ([]byte, error) {
			return environmentsSchema, nil
		}),

		// Step 2 — expand ${VAR} placeholders from OS environment.
		synthra.WithEnvSubst(synthra.FromEnv()),

		// Step 3 — validate the fully-substituted manifest against the
		// complete schema, which includes "service" (may have been a
		// placeholder before substitution).
		synthra.WithJSONSchemaFunc(func(_ *synthra.Values) ([]byte, error) {
			return manifestSchema, nil
		}),
	)
	if err != nil {
		log.Fatalf("new: %v", err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		var ce *synthra.ConfigError
		if errors.As(err, &ce) {
			log.Fatalf("load failed at %s: %v", ce.Path, ce.Err)
		}
		log.Fatalf("load: %v", err)
	}

	apiVersion, err := cfg.String("apiversion")
	if err != nil {
		log.Fatalf("apiversion: %v", err)
	}
	service, err := cfg.String("service")
	if err != nil {
		log.Fatalf("service: %v", err)
	}
	port, err := cfg.Int("port")
	if err != nil {
		log.Fatalf("port: %v", err)
	}

	fmt.Printf("apiVersion=%s  service=%s  port=%d\n", apiVersion, service, port)

	// Print environments
	envs, ok := cfg.Get("environments").([]any)
	if !ok {
		fmt.Println("environments: (none)")
		return
	}
	for i, e := range envs {
		envMap, isMap := e.(map[string]any)
		if !isMap {
			continue
		}
		fmt.Printf("environments[%d]: name=%v  envFile=%v\n", i, envMap["name"], envMap["envfile"])
	}
}
