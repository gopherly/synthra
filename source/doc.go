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

// Package source provides configuration source implementations.
//
// Types here load data from various locations and satisfy
// [gopherly.dev/synthra.Source] for use with [gopherly.dev/synthra.WithSource]
// and related options.
//
// # Available Sources
//
//   - File: Load configuration from the host file system
//   - FileFS: Load configuration from a path inside an [io/fs.FS]
//   - Map: In-memory map (defaults, embedded trees, tests)
//   - OSEnvVar: Load configuration from environment variables
//   - Consul: Load configuration from Consul key-value store
//
// # Example
//
// Creating a file source:
//
//	fileSource := source.NewFile("config.yaml", codec.YAML)
//	config, err := fileSource.Load(context.Background())
//
// Creating an environment variable source:
//
//	envSource := source.NewOSEnvVar("APP_")
//	config, err := envSource.Load(context.Background())
//
// Loading from an [io/fs.FS] (for example [testing/fstest.MapFS] or [embed.FS]):
//
//	fsys := fstest.MapFS{"cfg.yaml": &fstest.MapFile{Data: []byte("k: v\n")}}
//	fsSrc := source.NewFileFS(fsys, "cfg.yaml", codec.YAML)
//	config, err := fsSrc.Load(context.Background())
package source
