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

// Package main demonstrates Synthra's case-preserving, case-insensitive merge
// and shows how a JSON Schema acts as the canonical authority for key casing.
//
// Without a schema:
//
//	config-base.yaml has  ApiVersion: v1  (mixed case)
//	config-override.yaml has  apiVersion: v2  (canonical casing)
//	Result: ApiVersion: v2  — first source's casing wins; value overridden
//
// With a JSON Schema declaring "apiVersion":
//
//	Synthra renames ApiVersion -> apiVersion before validation runs.
//	Result: apiVersion: v2  — schema is the authority for casing.
//
// The logLevel key also demonstrates the same pattern: the base file uses
// "INFO" (uppercase value, preserved) and the override uses "warn".
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

	// Without schema: first source's casing for key names wins.
	noSchema, err := synthra.New(
		synthra.WithFile("config-base.yaml"),
		synthra.WithFile("config-override.yaml"),
	)
	if err != nil {
		log.Fatalf("new (no schema): %v", err)
	}
	if err = noSchema.Load(context.Background()); err != nil {
		log.Fatalf("load (no schema): %v", err)
	}

	// Both casings resolve to the same value through case-insensitive access.
	v1, err := noSchema.String("ApiVersion")
	if err != nil {
		log.Fatalf("get ApiVersion: %v", err)
	}
	v2, err := noSchema.String("apiVersion")
	if err != nil {
		log.Fatalf("get apiVersion: %v", err)
	}
	fmt.Printf("no schema — ApiVersion=%q  apiVersion=%q  (same value, first-source casing preserved)\n", v1, v2)

	// With schema: keys are renamed to match schema's "properties" declarations.
	withSchema, err := synthra.New(
		synthra.WithFile("config-base.yaml"),
		synthra.WithFile("config-override.yaml"),
		synthra.WithJSONSchema(schema),
	)
	if err != nil {
		log.Fatalf("new (with schema): %v", err)
	}
	if err = withSchema.Load(context.Background()); err != nil {
		log.Fatalf("load (with schema): %v", err)
	}

	apiVer, err := withSchema.String("apiVersion")
	if err != nil {
		log.Fatalf("get apiVersion: %v", err)
	}
	logLvl, err := withSchema.String("logLevel")
	if err != nil {
		log.Fatalf("get logLevel: %v", err)
	}
	svc, err := withSchema.String("service")
	if err != nil {
		log.Fatalf("get service: %v", err)
	}
	fmt.Printf("with schema — apiVersion=%q  logLevel=%q  service=%q\n", apiVer, logLvl, svc)
}
