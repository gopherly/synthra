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

// Package synthratest provides test helpers for packages that import
// [gopherly.dev/synthra].
//
// It depends on [github.com/stretchr/testify/require] for concise failures.
// Helpers that take [testing.T] call [testing.T.Helper] so failures report
// the caller’s line.
//
// # Map-backed configuration
//
// Use [Load] to prepend a [gopherly.dev/synthra/source.Map] source and load
// with [testing.T.Context]:
//
//	func TestApp(t *testing.T) {
//	    t.Parallel()
//	    cfg := synthratest.Load(t, map[string]any{
//	        "app":  "demo",
//	        "port": 8080,
//	    })
//	    synthratest.AssertString(t, cfg, "app", "demo")
//	    synthratest.AssertInt(t, cfg, "port", 8080)
//	}
//
// # File-backed configuration
//
// Use [WriteFile] and [LoadFile] to exercise real decoding and paths:
//
//	func TestFromYAML(t *testing.T) {
//	    t.Parallel()
//	    yaml := []byte("service:\n  name: api\n")
//	    cfg := synthratest.LoadFile(t, synthratest.YAML, yaml)
//	    synthratest.AssertString(t, cfg, "service.name", "api")
//	}
//
// [Format] values are [YAML], [JSON], and [TOML] for the temp file extension.
//
// # Deferred load
//
// Use [Config] when you need a [*synthra.Synthra] but want a custom
// [context.Context] or load sequence:
//
//	func TestCustomContext(t *testing.T) {
//	    t.Parallel()
//	    cfg := synthratest.Config(t, synthra.WithSource(source.NewMap(map[string]any{
//	        "k": "v",
//	    })))
//	    require.NoError(t, cfg.Load(context.Background()))
//	    synthratest.AssertString(t, cfg, "k", "v")
//	}
//
// # Assertions
//
// [AssertString], [AssertInt], [AssertBool], and [AssertStringSlice] wrap
// [github.com/stretchr/testify/require] with typed [synthra.Synthra] getters.
//
// # Test doubles
//
// [ErrSource] returns a [synthra.Source] that always fails with a fixed error.
// [Dumper] records [synthra.Synthra.Dump] calls; use [AssertDumped] to assert
// one call and the last map. [FuncCodec] is a minimal
// [gopherly.dev/synthra/codec.Decoder] and [gopherly.dev/synthra/codec.Encoder]
// backed by functions for table-driven codec tests.
//
// Runnable examples for the doubles and [FuncCodec] live in example_test.go.
package synthratest
