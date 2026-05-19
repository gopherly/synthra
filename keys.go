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

import "strings"

// findKeyFold returns the actual key in m that case-insensitively equals key,
// or "" if no such key exists. Exact match is checked first so the common
// case stays O(1).
func findKeyFold(m map[string]any, key string) string {
	if _, ok := m[key]; ok {
		return key
	}
	for k := range m {
		if strings.EqualFold(k, key) {
			return k
		}
	}
	return ""
}

// canonicalizeSchemaKeys renames case-different keys in values to match the
// casing of the schema's "properties" declarations. It recurses into nested
// objects and into array elements when the schema declares "items".
//
// Keys not declared in the schema are left alone. patternProperties and
// additionalProperties dynamic keys are not renamed.
func canonicalizeSchemaKeys(values, schema map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return values
	}
	for canonicalKey, propSchemaRaw := range props {
		var propSchema map[string]any
		if m, isPropMap := propSchemaRaw.(map[string]any); isPropMap {
			propSchema = m
		}

		actualKey := findKeyFold(values, canonicalKey)
		if actualKey != "" && actualKey != canonicalKey {
			values[canonicalKey] = values[actualKey]
			delete(values, actualKey)
		}
		if propSchema == nil {
			continue
		}
		switch v := values[canonicalKey].(type) {
		case map[string]any:
			values[canonicalKey] = canonicalizeSchemaKeys(v, propSchema)
		case []any:
			if itemSchema, itemSchemaOK := propSchema["items"].(map[string]any); itemSchemaOK {
				for i, elem := range v {
					if elemMap, elemMapOK := elem.(map[string]any); elemMapOK {
						v[i] = canonicalizeSchemaKeys(elemMap, itemSchema)
					}
				}
			}
		}
	}
	return values
}
