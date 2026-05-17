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

// Package resolve provides variable resolvers for use with
// [gopherly.dev/synthra.WithEnvSubst].
//
// A [Resolver] looks up a variable name and returns its value. You
// can build resolvers from static maps, OS environment variables, or
// your own lookup logic. Use [Chain] to combine multiple resolvers
// with priority ordering (last resolver wins).
//
// # Available Resolvers
//
//   - [Vars]: look up variables from a map[string]string
//   - [OS]: look up variables from [os.LookupEnv]
//   - [OSPrefix]: look up OS env vars that match a prefix (prefix is stripped)
//   - [Chain]: combine multiple resolvers (last wins)
//
// # How Priority Works
//
// When you pass multiple resolvers to [Chain] or to
// [gopherly.dev/synthra.WithEnvSubst], they are checked in order. If
// more than one resolver knows the same variable, the last one in the
// list wins. This means you can put lower-priority defaults first and
// higher-priority overrides last.
//
// # Example
//
// Combine a static map with OS environment overrides:
//
//	r := resolve.Chain(
//	    resolve.Vars(map[string]string{"PORT": "3000"}),
//	    resolve.OS(),
//	)
//	val, ok := r("PORT")
//	// If PORT is set in the environment, ok is true and val is the
//	// OS value. Otherwise val is "3000" from the static map.
//
// Use with WithEnvSubst:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(
//	        resolve.Vars(defaults),     // lowest priority
//	        resolve.Vars(fileVars),     // medium priority
//	        resolve.OSPrefix("APP_"),   // highest priority
//	    ),
//	)
package resolve
