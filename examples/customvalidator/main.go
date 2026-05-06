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

// Package main demonstrates cross-field checks with synthra.WithValidator.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"gopherly.dev/synthra"
)

func tlsPathsConsistent(m map[string]any) error {
	server, ok := m["server"].(map[string]any)
	if !ok {
		return nil
	}
	tls, ok := server["tls"].(map[string]any)
	if !ok {
		return nil
	}
	enabled := strings.EqualFold(fmt.Sprint(tls["enabled"]), "true")
	if !enabled {
		return nil
	}
	cert := nestedString(tls, "cert", "file")
	key := nestedString(tls, "key", "file")
	if cert == "" || key == "" {
		return errors.New("when server.tls.enabled is true, server.tls.cert.file and server.tls.key.file must be set")
	}
	return nil
}

func nestedString(m map[string]any, a, b string) string {
	x, ok := m[a].(map[string]any)
	if !ok {
		return ""
	}
	v, ok := x[b].(string)
	if !ok {
		return fmt.Sprint(x[b])
	}
	return strings.TrimSpace(v)
}

func main() {
	path := "config.yaml"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	cfg, err := synthra.New(
		synthra.WithFile(path),
		synthra.WithValidator(tlsPathsConsistent),
	)
	if err != nil {
		log.Fatalf("new: %v", err)
	}

	if err = cfg.Load(context.Background()); err != nil {
		log.Fatalf("load: %v", err)
	}

	fmt.Println("TLS configuration is consistent.")
}
