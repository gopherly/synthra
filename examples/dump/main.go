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

// Package main writes the merged effective configuration to a YAML file.
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
		synthra.WithFile("config.yaml"),
		synthra.WithEnv("APP_"),
		synthra.WithFileDumperAs(out, codec.YAML),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}
	if err := cfg.Dump(context.Background()); err != nil {
		log.Fatalf("dump: %v", err)
	}
	fmt.Printf("Wrote merged configuration to %s\n", out)
}
