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

package resolve

import (
	"os"
)

// Resolver looks up a variable by name. It returns the value and
// whether the variable was found. A nil Resolver always returns
// ("", false). A Resolver must be safe for concurrent use.
type Resolver func(name string) (value string, found bool)

// Vars returns a Resolver that looks up variables from a static map.
// The map is read as-is; keys are case-sensitive.
// Passing a nil map returns a Resolver that never finds any variable.
//
// Example:
//
//	r := resolve.Vars(map[string]string{
//	    "ENV":  "production",
//	    "PORT": "8080",
//	})
//	val, ok := r("PORT") // val = "8080", ok = true
func Vars(m map[string]string) Resolver {
	return func(name string) (string, bool) {
		v, ok := m[name]
		return v, ok
	}
}

// OS returns a Resolver that looks up variables using [os.LookupEnv].
// Each call reads the live process environment, so changes made
// between [gopherly.dev/synthra.Synthra.Load] calls are visible.
//
// Example:
//
//	r := resolve.OS()
//	val, ok := r("HOME") // looks up os.LookupEnv("HOME")
func OS() Resolver {
	return os.LookupEnv
}

// OSPrefix returns a Resolver that looks up OS environment variables
// with the given prefix. The prefix is stripped before matching, so
// the caller asks for the short name and the resolver maps it to the
// prefixed environment variable.
//
// For example, OSPrefix("APP_") resolves "PORT" by looking up
// "APP_PORT" in the environment. If APP_PORT is set, the resolver
// returns its value. If it is not set, found is false.
//
// Example:
//
//	r := resolve.OSPrefix("APP_")
//	val, ok := r("PORT") // looks up os.LookupEnv("APP_PORT")
func OSPrefix(prefix string) Resolver {
	return func(name string) (string, bool) {
		return os.LookupEnv(prefix + name)
	}
}

// Chain merges multiple resolvers into one. Resolvers are checked in
// order; when more than one resolves the same variable name, the last
// one wins (highest priority last).
//
// If no resolvers are given, the returned Resolver always returns
// ("", false).
//
// Example:
//
//	r := resolve.Chain(
//	    resolve.Vars(map[string]string{"PORT": "3000"}),
//	    resolve.OS(),
//	)
//	// If PORT is set in the environment, OS() wins.
//	// Otherwise Vars() provides "3000" as a fallback.
func Chain(resolvers ...Resolver) Resolver {
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
