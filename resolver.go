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
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/hashicorp/go-envparse"
)

// Resolver looks up a variable by name for ${VAR} expansion.
// It follows the Go "comma ok" idiom, matching [os.LookupEnv]:
// the second return value reports whether the variable was found.
// A nil Resolver always returns ("", false).
// A Resolver must be safe for concurrent use.
//
// To consult multiple sources, compose them with [Resolver.Or]:
// the first Resolver that reports found wins.
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
// Example — prefix applied to a .env file resolver, with OS env as fallback:
//
//	envFile, err := synthra.FromEnvFile(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(synthra.FromEnv().Prefix("APP_").Or(envFile.Prefix("APP_"))),
//	)
func (r Resolver) Prefix(prefix string) Resolver {
	if prefix == "" {
		return r
	}
	return func(name string) (string, bool) {
		return r(prefix + name)
	}
}

// Or returns a Resolver that tries the receiver first, then each fallback in
// order, stopping at the first one that reports found. This is the standard
// Go lookup-chain pattern: the same semantics as [context.Value], where the
// innermost (highest-priority) context shadows outer ones.
//
// Precedence rule: the receiver has highest priority; fallbacks are consulted
// in the order they are listed. The first Resolver to return found=true wins.
//
// Empty string counts as found. A resolver that returns ("", true) stops the
// Or chain — no further fallback resolver is consulted. This matches
// [context.Value] behavior: a deliberately-set empty value shadows later
// contexts.
//
// Note: the envsubst library processes ${VAR:-default} according to POSIX,
// which fires when the variable is unset OR empty. So even if Or stops the
// chain with ("", true), a template that uses ${VAR:-fallback} will still
// expand to "fallback". Use bare ${VAR} or ${VAR-fallback} (single dash, unset
// only) if you need to distinguish unset from explicitly empty.
//
// Nil fallbacks in the list are silently skipped, which makes it safe to pass
// a conditionally-built resolver without a guard:
//
//	r.Or(maybeNilResolver, synthra.FromEnv())
//
// A nil receiver is treated as a Resolver that never finds anything; fallbacks
// are still consulted in order.
//
// Or() with no arguments returns the receiver unchanged.
//
// Example — three-layer priority (OS env prefix > .env file > static defaults):
//
//	envFile, err := synthra.FromEnvFile(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(
//	        synthra.FromEnv().Prefix("APP_").     // highest: prefixed OS env
//	            Or(envFile).                       // middle:  .env file
//	            Or(synthra.FromMap(defaults)),     // lowest:  static defaults
//	    ),
//	)
func (r Resolver) Or(fallbacks ...Resolver) Resolver {
	if len(fallbacks) == 0 {
		return r
	}
	return func(name string) (string, bool) {
		if r != nil {
			if v, ok := r(name); ok {
				return v, true
			}
		}
		for _, fb := range fallbacks {
			if fb == nil {
				continue
			}
			if v, ok := fb(name); ok {
				return v, true
			}
		}
		return "", false
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
//	    synthra.WithEnvSubst(synthra.FromEnv().Or(envFile)),
//	)
//
// Example — layered with .env.local override (first match wins via Or):
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
//	    synthra.WithEnvSubst(synthra.FromEnv().Or(local).Or(base)),
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

// FromEnvFileIfExists returns a Resolver backed by the .env file at path.
// If the file does not exist, a no-op Resolver and nil error are returned.
// Parse errors on an existing file are returned unchanged.
//
// Use this instead of [FromEnvFile] when the .env file is optional. To fall
// back across multiple candidates in order, see [CoalesceEnvFile].
//
// Example — OS env takes priority, .env is optional:
//
//	r, err := synthra.FromEnvFileIfExists(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(synthra.FromEnv().Or(r)),
//	)
func FromEnvFileIfExists(path string) (Resolver, error) {
	r, err := FromEnvFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return func(string) (string, bool) { return "", false }, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// CoalesceEnvFile returns a Resolver backed by the first existing path in the
// list. Missing paths are silently skipped. A parse error on a found file is
// returned and halts the search. With no paths or all paths missing, a no-op
// Resolver and nil error are returned.
//
// Follows SQL COALESCE semantics: first non-missing argument wins. To compose
// the returned Resolver with other sources, use [Resolver.Or].
//
// Example — try environment-specific file, then shared .env:
//
//	r, err := synthra.CoalesceEnvFile(".env."+env, ".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(synthra.FromEnv().Or(r)),
//	)
func CoalesceEnvFile(paths ...string) (Resolver, error) {
	for _, p := range paths {
		r, err := FromEnvFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return r, nil
	}
	return func(string) (string, bool) { return "", false }, nil
}
