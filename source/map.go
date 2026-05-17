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

package source

import "context"

// Map is a Source backed by an in-memory map.
//
// NewMap aliases the caller's map for efficiency; callers must not mutate
// that map after passing it to NewMap if Load may run concurrently or if
// they rely on a stable snapshot. To pass an independent copy, use
// [maps.Clone] (or another deep copy strategy) before calling NewMap.
//
// Each successful Load returns the same map reference.
type Map struct {
	conf map[string]any
}

// NewMap returns a Source that always loads from m. A nil m is treated as empty.
func NewMap(m map[string]any) *Map {
	if m == nil {
		m = map[string]any{}
	}
	return &Map{conf: m}
}

// Load returns the configured map.
func (s *Map) Load(context.Context) (map[string]any, error) {
	return s.conf, nil
}
