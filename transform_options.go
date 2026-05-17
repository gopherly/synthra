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
	"strings"
)

// WithTransform registers a function that transforms the merged configuration
// values during Load. Transforms run after JSON Schema defaults are applied
// and before JSON Schema validation, in the order they were registered. This
// means the schema validator sees the final, transformed values.
//
// The function receives the current values map and must return the (possibly
// modified) values map. Returning a nil map is treated as an empty map.
// Returning an error aborts Load with a [*ConfigError] whose Path identifies
// the failing transform by its index ("transform[0]", "transform[1]", …).
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

// WithInterpolation registers a transform that replaces {key} placeholders in
// all string values with the corresponding value from vars. Placeholders for
// keys not present in vars are left unchanged.
//
// Interpolation runs after JSON Schema defaults are applied and before JSON
// Schema validation, so the validated values reflect the substituted strings.
//
// Example — substitute the environment name into file paths:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithInterpolation(map[string]string{
//	        "name":   "production",
//	        "region": "eu-west-1",
//	    }),
//	    synthra.WithJSONSchema(schema),
//	)
//	// If config.yaml contains:
//	//   envFile: ".env.{name}"
//	//   cluster: "{region}-cluster"
//	// After Load:
//	//   cfg.Get("envFile") => ".env.production"
//	//   cfg.Get("cluster") => "eu-west-1-cluster"
func WithInterpolation(vars map[string]string) Option {
	return WithTransform(func(values map[string]any) (map[string]any, error) {
		interpolateStrings(values, vars)
		return values, nil
	})
}

// interpolateStrings recursively walks values and replaces {key} placeholders
// in all string values using vars. Non-string values are left untouched.
func interpolateStrings(values map[string]any, vars map[string]string) {
	if len(vars) == 0 {
		return
	}
	for k, v := range values {
		switch val := v.(type) {
		case string:
			values[k] = replacePlaceholders(val, vars)
		case map[string]any:
			interpolateStrings(val, vars)
		case []any:
			interpolateSlice(val, vars)
		}
	}
}

// interpolateSlice applies placeholder substitution to every element in a
// slice, recursing into nested maps.
func interpolateSlice(slice []any, vars map[string]string) {
	for i, elem := range slice {
		switch val := elem.(type) {
		case string:
			slice[i] = replacePlaceholders(val, vars)
		case map[string]any:
			interpolateStrings(val, vars)
		case []any:
			interpolateSlice(val, vars)
		}
	}
}

// replacePlaceholders replaces all {key} occurrences in s with the
// corresponding value from vars. Unmatched placeholders are left as-is.
func replacePlaceholders(s string, vars map[string]string) string {
	if !strings.ContainsRune(s, '{') {
		return s
	}
	for key, val := range vars {
		s = strings.ReplaceAll(s, fmt.Sprintf("{%s}", key), val)
	}
	return s
}
