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
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// applySchemaDefaults fills missing keys in values from JSON Schema "default"
// declarations without overriding values that are already present.
func applySchemaDefaults(values, schema map[string]any) map[string]any {
	if values == nil {
		values = make(map[string]any)
	}

	// Apply defaults from "properties" (fixed key names).
	if props, ok := schema["properties"].(map[string]any); ok {
		for rawKey, propSchemaRaw := range props {
			propSchema, propSchemaOk := propSchemaRaw.(map[string]any)
			if !propSchemaOk {
				continue
			}

			// Normalize the key to lowercase so it matches the keys produced
			// by normalizeMapKeys during source loading.
			key := strings.ToLower(rawKey)

			existing, exists := values[key]
			if !exists {
				// Key is absent: set the default value if the schema declares one.
				if def, hasDefault := propSchema["default"]; hasDefault {
					values[key] = def
				}
				// If there's no default, still recurse for nested objects that
				// have their own defaults. Create an empty map only when the
				// property schema describes an object with properties.
				if _, isObj := propSchema["properties"]; isObj {
					nested := applySchemaDefaults(make(map[string]any), propSchema)
					if len(nested) > 0 {
						values[key] = nested
					}
				}
			} else {
				switch val := existing.(type) {
				case map[string]any:
					// Recurse into nested objects.
					values[key] = applySchemaDefaults(val, propSchema)
				case []any:
					// Recurse into array elements using the "items" schema.
					values[key] = applyItemDefaults(val, propSchema)
				}
			}
		}
	}

	// Apply defaults from "patternProperties" (regex-matched key names).
	// For each pattern, iterate over the actual keys in values and apply
	// defaults to every key whose name matches the pattern.
	if patternProps, ok := schema["patternProperties"].(map[string]any); ok {
		for pattern, patternSchemaRaw := range patternProps {
			patternSchema, patternSchemaOk := patternSchemaRaw.(map[string]any)
			if !patternSchemaOk {
				continue
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				// Skip invalid patterns rather than crashing; the JSON Schema
				// compiler would have already rejected them at construction
				// time, so this path is defensive only.
				continue
			}

			for key, val := range values {
				if !re.MatchString(key) {
					continue
				}

				// The key matches the pattern. Ensure the value is a map
				// (object) so we can apply nested defaults to it.
				existing, isMap := val.(map[string]any)
				if !isMap {
					existing = make(map[string]any)
				}
				values[key] = applySchemaDefaults(existing, patternSchema)
			}
		}
	}

	return values
}

// applyItemDefaults applies item-schema defaults to each map element of a slice.
func applyItemDefaults(slice []any, propSchema map[string]any) []any {
	itemSchema, ok := propSchema["items"].(map[string]any)
	if !ok {
		return slice
	}
	for i, elem := range slice {
		if elemMap, elemMapOk := elem.(map[string]any); elemMapOk {
			slice[i] = applySchemaDefaults(elemMap, itemSchema)
		}
	}
	return slice
}

// compileJSONSchema compiles raw JSON Schema bytes into a validated, executable
// schema and also returns the raw parsed map for default extraction.
// It is shared by [WithJSONSchema] (at construction time) and the
// [WithJSONSchemaSelector] path inside [Synthra.Load] (at load time).
func compileJSONSchema(schema []byte) (*jsonschema.Schema, map[string]any, error) {
	// Use a unique schema name to avoid caching issues across calls.
	//nolint:gosec // rand.Int() is used for a unique schema name, not security sensitive
	schemaName := fmt.Sprintf("inline_%d.json", rand.Int())
	compiler := jsonschema.NewCompiler()

	jsonSchema, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return nil, nil, err
	}
	addErr := compiler.AddResource(schemaName, jsonSchema)
	if addErr != nil {
		return nil, nil, addErr
	}
	compiled, err := compiler.Compile(schemaName)
	if err != nil {
		return nil, nil, err
	}

	var raw map[string]any
	unmarshalErr := json.Unmarshal(schema, &raw)
	if unmarshalErr != nil {
		return nil, nil, unmarshalErr
	}
	return compiled, raw, nil
}
