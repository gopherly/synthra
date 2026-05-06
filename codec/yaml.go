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

import "github.com/goccy/go-yaml"

// YAML is a Codec that encodes and decodes YAML.
var YAML Codec = yamlCodec{}

type yamlCodec struct{}

func (yamlCodec) Encode(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

func (yamlCodec) Decode(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}
