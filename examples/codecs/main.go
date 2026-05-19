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

// Package main shows codec usage end-to-end: JSON + TOML as input sources,
// merged YAML written out via WithFileDumperAs.
//
// Sources (lowest to highest priority):
//  1. app.json   — loaded with WithFileAs and codec.JSON
//  2. overrides.toml — loaded with WithFileAs and codec.TOML (wins on conflicts)
//
// After Load, Dump writes the merged state to effective-config.yaml.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gopherly.dev/synthra"
	"gopherly.dev/synthra/codec"
)

func main() {
	out := "effective-config.yaml"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	cfg := synthra.MustNew(
		synthra.WithFileAs("app.json", codec.JSON),
		synthra.WithFileAs("overrides.toml", codec.TOML),
		synthra.WithFileDumperAs(out, codec.YAML),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}
	if err := cfg.Dump(context.Background()); err != nil {
		log.Fatalf("dump: %v", err)
	}

	app, err := cfg.String("app")
	if err != nil {
		log.Fatalf("app: %v", err)
	}
	listenPort, err := cfg.Int("listen.port")
	if err != nil {
		log.Fatalf("listen.port: %v", err)
	}
	region, err := cfg.String("meta.region")
	if err != nil {
		log.Fatalf("meta.region: %v", err)
	}

	fmt.Printf("app=%s listen.port=%d meta.region=%s\n", app, listenPort, region)
	fmt.Printf("Wrote merged configuration to %s\n", out)
}
