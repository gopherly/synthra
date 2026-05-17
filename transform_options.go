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

	"github.com/fluxcd/pkg/envsubst"
	"gopherly.dev/synthra/resolve"
)

// WithTransform registers a function that transforms the merged configuration
// values during Load. Transforms run after JSON Schema defaults are applied
// and before JSON Schema validation, in the order they were registered. This
// means the schema validator sees the final, transformed values.
//
// The function receives the current values map and must return the (possibly
// modified) values map. Returning a nil map is treated as an empty map.
// Returning an error aborts Load with a [*ConfigError] whose Path identifies
// the failing transform by its index ("transform[0]", "transform[1]", ...).
//
// Multiple transforms are applied as a pipeline: the output of transform N
// becomes the input of transform N+1.
//
// Example — normalize log level to lowercase before validation:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithTransform(func(values map[string]any) (map[string]any, error) {
//	        if level, ok := values["log_level"].(string); ok {
//	            values["log_level"] = strings.ToLower(level)
//	        }
//	        return values, nil
//	    }),
//	    synthra.WithJSONSchema(schema),
//	)
func WithTransform(fn func(map[string]any) (map[string]any, error)) Option {
	return func(cfg *config) {
		if fn == nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithTransform", errors.New("transform function cannot be nil")))
			return
		}
		cfg.transforms = append(cfg.transforms, fn)
	}
}

// WithEnvSubst registers a transform that expands POSIX-style ${VAR}
// placeholders in all string values of the merged configuration map.
//
// Resolvers are consulted in order. When more than one resolver knows
// the same variable name, the last one wins (highest priority last).
// Use [resolve.Chain] to pre-compose resolvers if needed.
//
// Supported syntax includes ${VAR}, ${VAR:-default}, ${VAR:=default},
// ${VAR^^}, ${VAR#pattern}, and more. The full set is documented at
// https://pkg.go.dev/github.com/fluxcd/pkg/envsubst.
//
// This is different from [WithEnv]. [WithEnv] is a source: it reads
// environment variables and adds them as configuration keys. For
// example, APP_SERVER_PORT=8080 becomes server.port in the config
// map. [WithEnvSubst] is a transform: it expands ${VAR} placeholders
// that appear inside string values already loaded from other sources.
// Both can be used together without overlap.
//
// Example — expand placeholders using a static map:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(resolve.Vars(map[string]string{
//	        "ENV":  "production",
//	        "PORT": "8080",
//	    })),
//	)
//	// If config.yaml contains: host: "app-${ENV}.example.com"
//	// After Load: cfg.Get("host") => "app-production.example.com"
//
// Example — layer multiple resolvers for priority-ordered substitution:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("deployah.yaml"),
//	    synthra.WithEnvSubst(
//	        resolve.Vars(manifestVars),    // lowest priority
//	        resolve.Vars(envFileVars),     // medium priority
//	        resolve.OSPrefix("DPY_VAR_"),  // highest priority
//	    ),
//	)
//	// config.yaml: port: ${PORT:-3000}
//	// If DPY_VAR_PORT=9090 is set, port becomes "9090".
//	// If DPY_VAR_PORT is not set but PORT is in envFileVars, that wins.
//	// If neither is set, the ${VAR:-default} fallback gives "3000".
func WithEnvSubst(resolvers ...resolve.Resolver) Option {
	merged := resolve.Chain(resolvers...)
	return WithTransform(func(values map[string]any) (map[string]any, error) {
		if err := envsubstMap(values, merged, ""); err != nil {
			return nil, fmt.Errorf("envsubst: %w", err)
		}
		return values, nil
	})
}

// envsubstMap recursively walks values and expands ${VAR} placeholders
// in all string values using the mapping function. The prefix
// accumulates the dotted path for error messages.
func envsubstMap(values map[string]any, mapping func(string) (string, bool), prefix string) error {
	for k, v := range values {
		path := prefix + k
		switch val := v.(type) {
		case string:
			expanded, err := envsubst.Eval(val, mapping)
			if err != nil {
				return fmt.Errorf("key %q: %w", path, err)
			}
			values[k] = expanded
		case map[string]any:
			if err := envsubstMap(val, mapping, path+"."); err != nil {
				return err
			}
		case []any:
			if err := envsubstSlice(val, mapping, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// envsubstSlice applies envsubst expansion to every element in a slice,
// recursing into nested maps and slices. The prefix is the parent key
// path; indices are appended as [N].
func envsubstSlice(slice []any, mapping func(string) (string, bool), prefix string) error {
	for i, elem := range slice {
		path := fmt.Sprintf("%s[%d]", prefix, i)
		switch val := elem.(type) {
		case string:
			expanded, err := envsubst.Eval(val, mapping)
			if err != nil {
				return fmt.Errorf("key %q: %w", path, err)
			}
			slice[i] = expanded
		case map[string]any:
			if err := envsubstMap(val, mapping, path+"."); err != nil {
				return err
			}
		case []any:
			if err := envsubstSlice(val, mapping, path); err != nil {
				return err
			}
		}
	}
	return nil
}
