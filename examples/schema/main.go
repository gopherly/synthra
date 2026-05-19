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

// Package main demonstrates JSON Schema defaults and WithEnvSubst.
//
// config.yaml provides only required and non-default values. Synthra
// automatically fills in missing keys from the schema "default" declarations,
// including patternProperties defaults applied to every matching component.
// WithEnvSubst then expands ${ENV} placeholders in string values.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gopherly.dev/synthra"
)

func main() {
	schema, err := os.ReadFile("schema.json")
	if err != nil {
		log.Fatalf("read schema: %v", err)
	}

	// WithEnvSubst expands ${ENV} in any string value.
	// The config.yaml uses "${ENV}" in no field here, but you can add
	// e.g. "image: my-app:${ENV}" to see it in action.
	env := "production"

	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithJSONSchema(schema), // validates AND applies "default" values
		synthra.WithEnvSubst(synthra.FromMap(map[string]string{
			"ENV": env,
		})),
	)
	if err != nil {
		log.Fatalf("new: %v", err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}

	// Fields present in config.yaml
	svc, err := cfg.String("service")
	if err != nil {
		log.Fatalf("get service: %v", err)
	}
	maxConn, err := cfg.Int("server.maxConnections")
	if err != nil {
		log.Fatalf("get server.maxConnections: %v", err)
	}

	// Fields filled from JSON Schema "default"
	port, err := cfg.Int("port")
	if err != nil {
		log.Fatalf("get port: %v", err)
	}
	logLevel, err := cfg.String("logLevel")
	if err != nil {
		log.Fatalf("get logLevel: %v", err)
	}
	timeout, err := cfg.String("server.timeout")
	if err != nil {
		log.Fatalf("get server.timeout: %v", err)
	}

	fmt.Printf("service=%s  port=%d  log_level=%s\n", svc, port, logLevel)
	fmt.Printf("server: timeout=%s  max_connections=%d\n", timeout, maxConn)

	// patternProperties defaults applied to each component
	webRole, err := cfg.String("components.web.role")
	if err != nil {
		log.Fatalf("get components.web.role: %v", err)
	}
	webReplicas, err := cfg.Int("components.web.replicas")
	if err != nil {
		log.Fatalf("get components.web.replicas: %v", err)
	}
	workerRole, err := cfg.String("components.worker.role")
	if err != nil {
		log.Fatalf("get components.worker.role: %v", err)
	}
	workerReplicas, err := cfg.Int("components.worker.replicas")
	if err != nil {
		log.Fatalf("get components.worker.replicas: %v", err)
	}

	fmt.Printf("components.web:    role=%s  replicas=%d\n", webRole, webReplicas)
	fmt.Printf("components.worker: role=%s  replicas=%d\n", workerRole, workerReplicas)
}
