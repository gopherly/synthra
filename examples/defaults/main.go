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

// Package main shows merge order: baked-in defaults, then file, then env.
package main

import (
	"context"
	"fmt"
	"log"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/codec"
)

func main() {
	defaults := []byte(`
server:
  port: 3000
  name: "defaults-only"
`)

	cfg := synthra.MustNew(
		synthra.WithContent(defaults, codec.YAML),
		synthra.WithFile("overrides.yaml"),
		synthra.WithEnv("DEMO_"),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}

	name, err := cfg.String("server.name")
	if err != nil {
		log.Fatalf("server.name: %v", err)
	}
	port, err := cfg.Int("server.port")
	if err != nil {
		log.Fatalf("server.port: %v", err)
	}
	fmt.Printf("server.name=%s server.port=%d\n", name, port)
}
