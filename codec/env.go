// Copyright 2026 The Gopherly Authors
// Copyright 2025 Company.info B.V.
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

package codec

import (
	"bytes"
	"fmt"
	"strings"
)

// EnvVar is a Decoder that decodes the environment variable format.
var EnvVar Decoder = envVarCodec{}

type envVarCodec struct{}

func (envVarCodec) Decode(data []byte, v any) error {
	conf := make(map[string]any)

	for env := range bytes.SplitSeq(data, []byte("\n")) {
		pair := strings.SplitN(string(env), "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := strings.TrimSpace(pair[0])
		if key == "" {
			continue
		}

		rawParts := strings.Split(strings.ToLower(key), "_")
		parts := make([]string, 0, len(rawParts))
		for _, part := range rawParts {
			if part != "" {
				parts = append(parts, part)
			}
		}

		if len(parts) == 0 {
			continue
		}

		current := conf
		for i := 0; i < len(parts)-1; i++ {
			part := parts[i]
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]any)
			}
			if nextMap, ok := current[part].(map[string]any); ok {
				current = nextMap
			} else {
				current[part] = make(map[string]any)
				if innerMap, okInner := current[part].(map[string]any); okInner {
					current = innerMap
				} else {
					return fmt.Errorf("failed to create nested map for key: %s", part)
				}
			}
		}

		current[parts[len(parts)-1]] = strings.TrimSpace(pair[1])
	}

	ptr, ok := v.(*map[string]any)
	if !ok {
		return fmt.Errorf("envVarCodec.Decode: expected *map[string]any, got %T", v)
	}
	*ptr = conf

	return nil
}
