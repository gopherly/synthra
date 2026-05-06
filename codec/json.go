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

import "encoding/json"

// JSON is a Codec that encodes and decodes JSON.
var JSON Codec = jsonCodec{}

type jsonCodec struct{}

func (jsonCodec) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
