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

package synthra

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-envparse"
)

// Resolver looks up a variable by name for ${VAR} expansion.
// It follows the Go "comma ok" idiom, matching [os.LookupEnv]:
// the second return value reports whether the variable was found.
// A nil Resolver always returns ("", false).
// A Resolver must be safe for concurrent use.
type Resolver func(name string) (value string, found bool)

// FromMap returns a Resolver that looks up variables from a static map.
// The map is read as-is; keys are case-sensitive.
// Passing a nil map returns a Resolver that never finds any variable.
//
// Example:
//
//	r := synthra.FromMap(map[string]string{
//	    "ENV":  "production",
//	    "PORT": "8080",
//	})
func FromMap(m map[string]string) Resolver {
	return func(name string) (string, bool) {
		v, ok := m[name]
		return v, ok
	}
}

// FromEnv returns a Resolver that reads from the live OS environment
// using [os.LookupEnv]. Changes between [Synthra.Load] calls are visible.
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(synthra.FromEnv()),
//	)
func FromEnv() Resolver {
	return os.LookupEnv
}

// Prefix returns a Resolver that prepends prefix to every lookup name
// before delegating to the receiver. Prefix("APP_") resolves "PORT" by
// looking up "APP_PORT" in the underlying resolver. This works with any
// Resolver — [FromEnv], [FromEnvFile], or [FromMap].
//
// If prefix is empty, the receiver is returned unchanged.
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(synthra.FromEnv().Prefix("APP_")),
//	)
//
// Example — prefix applied to a .env file resolver:
//
//	envFile, err := synthra.FromEnvFile(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(envFile.Prefix("APP_"), synthra.FromEnv().Prefix("APP_")),
//	)
func (r Resolver) Prefix(prefix string) Resolver {
	if prefix == "" {
		return r
	}
	return func(name string) (string, bool) {
		return r(prefix + name)
	}
}

// FromEnvFile reads a .env file eagerly and returns a map-backed Resolver.
// The file is read and parsed at call time; if the file does not exist or
// cannot be parsed, an error is returned.
//
// Parsing is handled by [github.com/hashicorp/go-envparse] which supports
// quoted values, comment lines, the export prefix, inline comments, and
// escape sequences.
//
// Supported syntax:
//   - KEY=VALUE (simple assignment)
//   - KEY="VALUE" or KEY='VALUE' (quoted values, preserves whitespace)
//   - # comment lines (ignored)
//   - export KEY=VALUE (export prefix stripped)
//   - Empty lines (ignored)
//
// Example:
//
//	envFile, err := synthra.FromEnvFile(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(envFile, synthra.FromEnv()),
//	)
//
// Example — layered with .env.local override (last wins):
//
//	base, err := synthra.FromEnvFile(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	local, err := synthra.FromEnvFile(".env.local")
//	if err != nil && !os.IsNotExist(err) {
//	    log.Fatal(err)
//	}
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(base, local, synthra.FromEnv()),
//	)
func FromEnvFile(path string) (Resolver, error) {
	f, err := os.Open(path) //nolint:gosec // caller controls the path
	if err != nil {
		return nil, fmt.Errorf("FromEnvFile(%q): %w", path, err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // closing a read-only file cannot fail meaningfully

	m, err := envparse.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("FromEnvFile(%q): %w", path, err)
	}

	return FromMap(m), nil
}

// chainResolvers merges multiple resolvers into one. Resolvers are checked in
// order; when more than one resolves the same variable name, the last one wins
// (highest priority last). Nil resolvers in the chain are skipped.
//
// If no resolvers are given, the returned Resolver always returns ("", false).
func chainResolvers(resolvers ...Resolver) Resolver {
	return func(name string) (string, bool) {
		var val string
		var found bool
		for _, r := range resolvers {
			if r == nil {
				continue
			}
			if v, ok := r(name); ok {
				val = v
				found = true
			}
		}
		return val, found
	}
}
