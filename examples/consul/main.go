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

// Package main loads local YAML, optionally merges Consul KV, then env.
//
// When CONSUL_HTTP_ADDR is unset, WithIf adds no Consul source
// and the program still runs using file + environment only.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gopherly.dev/synthra"
)

func main() {
	cfg := synthra.MustNew(
		synthra.WithFile("config.yaml"),
		synthra.WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "", synthra.WithConsul("synthra/example/config.yaml")),
		synthra.WithEnv("EDGE_"),
	)

	if err := cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}

	svcName, err := cfg.String("service.name")
	if err != nil {
		log.Fatalf("service.name: %v", err)
	}
	svcPort, err := cfg.Int("service.port")
	if err != nil {
		log.Fatalf("service.port: %v", err)
	}
	fmt.Printf("service.name=%s service.port=%d\n", svcName, svcPort)
}
