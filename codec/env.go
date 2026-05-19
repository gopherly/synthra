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
	"sort"
	"strings"

	"github.com/hashicorp/go-envparse"
)

// EnvVar is a Decoder that decodes the environment variable format.
var EnvVar Decoder = envVarCodec{}

type envVarCodec struct{}

func (envVarCodec) Decode(data []byte, v any) error {
	pairs, err := envparse.Parse(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("envVarCodec.Decode: %w", err)
	}

	// Sort keys so shorter entries (e.g. A=scalar) are always processed before
	// longer nested ones (e.g. A_B=nested). This guarantees that nested keys
	// consistently overwrite scalars regardless of map iteration order.
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	conf := make(map[string]any)
	for _, key := range keys {
		val := pairs[key]
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
				nested := make(map[string]any)
				current[part] = nested
				current = nested
			}
		}

		current[parts[len(parts)-1]] = val
	}

	ptr, ok := v.(*map[string]any)
	if !ok {
		return fmt.Errorf("envVarCodec.Decode: expected *map[string]any, got %T", v)
	}
	*ptr = conf

	return nil
}
