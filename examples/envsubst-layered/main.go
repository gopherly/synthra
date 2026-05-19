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

// Package main demonstrates three-layer resolver composition with Resolver.Or.
//
// Variable precedence (first match wins):
//
//  1. Prefixed OS environment (DPY_VAR_*) — highest priority, set by the operator
//  2. A .env file — medium priority, committed per-environment defaults
//  3. Static manifest defaults — lowest priority, fallback values embedded in code
//
// This pattern is typical for deploy tools (Deployah-style) where:
//   - the operator overrides individual variables at runtime via a prefix
//   - per-environment .env files carry most of the values
//   - hardcoded defaults cover the rest
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gopherly.dev/synthra"
)

// manifestDefaults are the lowest-priority fallback values baked into the
// manifest. Any variable not found in the OS env or the .env file falls back
// here.
var manifestDefaults = map[string]string{
	"REGION":  "eu-west-1",
	"TAG":     "stable",
	"DB_HOST": "db.internal",
}

func main() {
	// Load an optional .env file. The path can be overridden via DPY_VAR_ENVFILE
	// or the ENVFILE key in manifestDefaults.
	envFilePath := ".env"
	if v := os.Getenv("DPY_VAR_ENVFILE"); v != "" {
		envFilePath = v
	}

	envFile, err := synthra.FromEnvFile(envFilePath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("load env file: %v", err)
	}
	if os.IsNotExist(err) {
		// .env is optional; use a resolver that never finds anything.
		envFile = synthra.FromMap(nil)
	}

	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithEnvSubst(
			// Precedence expressed in code, no comment needed:
			synthra.FromEnv().Prefix("DPY_VAR_"). // highest: DPY_VAR_* OS env
								Or(envFile).                           // middle:  .env file
								Or(synthra.FromMap(manifestDefaults)), // lowest:  static defaults
		),
	)
	if err != nil {
		log.Fatalf("new: %v", err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}

	service, err := cfg.String("service")
	if err != nil {
		log.Fatalf("get service: %v", err)
	}
	region, err := cfg.String("region")
	if err != nil {
		log.Fatalf("get region: %v", err)
	}
	image, err := cfg.String("image")
	if err != nil {
		log.Fatalf("get image: %v", err)
	}
	dbURL, err := cfg.String("db_url")
	if err != nil {
		log.Fatalf("get db_url: %v", err)
	}

	fmt.Printf("service=%s  region=%s\n", service, region)
	fmt.Printf("image=%s\n", image)
	fmt.Printf("db_url=%s\n", dbURL)
}
