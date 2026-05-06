// Copyright 2026 The Gopherly Authors
// Copyright 2025 Company.info B.V.
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

// Package main shows YAML file loading and struct binding with Synthra.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"gopherly.dev/synthra"
)

// Config is the configuration for the application.
type Config struct {
	Foo     string        `synthra:"foo"`
	Timeout time.Duration `synthra:"timeout"`
	Debug   bool          `synthra:"debug"`
	Worker  Worker        `synthra:"worker"`
	Date    time.Time     `synthra:"date"`
	Roles   []string      `synthra:"roles"`
	Types   []string      `synthra:"types"`
	Types2  string        `synthra:"types"`
}

// Worker is the worker configuration.
type Worker struct {
	Timeout time.Duration `synthra:"timeout"`
	Address *url.URL      `synthra:"address"`
}

// main is the main function.
func main() {
	var cfg Config

	c := synthra.MustNew(
		synthra.WithFile("./config.yaml"),
		synthra.WithBinding(&cfg),
	)

	err := c.Load(context.Background())
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Printf("%+v\n", cfg)
}
