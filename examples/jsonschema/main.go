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

// Package main demonstrates JSON Schema validation on loaded configuration.
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

	cfg, err := synthra.New(
		synthra.WithFile("config.yaml"),
		synthra.WithJSONSchema(schema),
	)
	if err != nil {
		log.Fatalf("new: %v", err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}

	svc, err := cfg.String("service")
	if err != nil {
		log.Fatalf("service: %v", err)
	}
	port, err := cfg.Int("port")
	if err != nil {
		log.Fatalf("port: %v", err)
	}
	level, err := cfg.String("log_level")
	if err != nil {
		log.Fatalf("log_level: %v", err)
	}
	fmt.Printf("service=%s port=%d log_level=%s\n", svc, port, level)
}
