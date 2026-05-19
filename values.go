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
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// Values wraps the merged configuration map for pipeline callbacks. Every
// read and write is case-insensitive at each path segment. Dot notation
// addresses nested maps.
//
// Values is not safe for concurrent use. Each Load call creates a fresh
// instance for the duration of the pipeline.
type Values struct {
	m map[string]any
}

// newValues wraps an existing map. The wrapped map is mutated in place.
func newValues(m map[string]any) *Values {
	if m == nil {
		m = make(map[string]any)
	}
	return &Values{m: m}
}

// Get returns the raw value at path or nil if no key matches. Lookup is
// case-insensitive at each dot-separated segment. A top-level key that
// literally contains a dot is checked first before falling back to
// dot-notation traversal.
func (v *Values) Get(path string) any {
	if k := findKeyFold(v.m, path); k != "" {
		return v.m[k]
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return nil
	}
	current := v.m
	for _, seg := range segments[:len(segments)-1] {
		k := findKeyFold(current, seg)
		if k == "" {
			return nil
		}
		nested, ok := current[k].(map[string]any)
		if !ok {
			return nil
		}
		current = nested
	}
	k := findKeyFold(current, segments[len(segments)-1])
	if k == "" {
		return nil
	}
	return current[k]
}

// Has reports whether path exists in the values.
func (v *Values) Has(path string) bool {
	if findKeyFold(v.m, path) != "" {
		return true
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return false
	}
	current := v.m
	for _, seg := range segments[:len(segments)-1] {
		k := findKeyFold(current, seg)
		if k == "" {
			return false
		}
		nested, ok := current[k].(map[string]any)
		if !ok {
			return false
		}
		current = nested
	}
	return findKeyFold(current, segments[len(segments)-1]) != ""
}

// Set assigns value at path. Intermediate maps are created as needed.
// Returns an error if a non-final segment along the path exists but is
// not a map.
func (v *Values) Set(path string, value any) error {
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		k := findKeyFold(v.m, path)
		if k == "" {
			k = path
		}
		v.m[k] = value
		return nil
	}
	current := v.m
	for _, seg := range segments[:len(segments)-1] {
		k := findKeyFold(current, seg)
		if k == "" {
			k = seg
			current[k] = make(map[string]any)
		}
		nested, ok := current[k].(map[string]any)
		if !ok {
			return fmt.Errorf("synthra: Set %q: segment %q is not a map", path, seg)
		}
		current = nested
	}
	last := segments[len(segments)-1]
	k := findKeyFold(current, last)
	if k == "" {
		k = last
	}
	current[k] = value
	return nil
}

// Delete removes the value at path. Reports whether something was removed.
func (v *Values) Delete(path string) bool {
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		k := findKeyFold(v.m, path)
		if k == "" {
			return false
		}
		delete(v.m, k)
		return true
	}
	current := v.m
	for _, seg := range segments[:len(segments)-1] {
		k := findKeyFold(current, seg)
		if k == "" {
			return false
		}
		nested, ok := current[k].(map[string]any)
		if !ok {
			return false
		}
		current = nested
	}
	k := findKeyFold(current, segments[len(segments)-1])
	if k == "" {
		return false
	}
	delete(current, k)
	return true
}

// Keys returns the top-level keys in their stored casing.
func (v *Values) Keys() []string {
	keys := make([]string, 0, len(v.m))
	for k := range v.m {
		keys = append(keys, k)
	}
	return keys
}

// Walk visits every node in the tree depth-first. The callback receives the
// dot-path to the node and its current value. Returning (newValue, true)
// replaces the node; returning (_, false) leaves it unchanged. Slice indices
// appear as "key[N]" in the path.
func (v *Values) Walk(fn func(path string, value any) (any, bool)) {
	walkMap(v.m, "", fn)
}

func walkMap(m map[string]any, prefix string, fn func(path string, value any) (any, bool)) {
	for k, val := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if newVal, replace := fn(path, val); replace {
			m[k] = newVal
			val = newVal
		}
		switch child := val.(type) {
		case map[string]any:
			walkMap(child, path, fn)
		case []any:
			walkSlice(child, path, fn)
		}
	}
}

func walkSlice(s []any, prefix string, fn func(path string, value any) (any, bool)) {
	for i, elem := range s {
		path := fmt.Sprintf("%s[%d]", prefix, i)
		if newVal, replace := fn(path, elem); replace {
			s[i] = newVal
			elem = newVal
		}
		switch child := elem.(type) {
		case map[string]any:
			walkMap(child, path, fn)
		case []any:
			walkSlice(child, path, fn)
		}
	}
}

// Raw returns the underlying map. Direct map access is case-sensitive at the
// Go level; use Raw only when you must hand the data to code that expects a
// plain map[string]any. Mutations on the returned map remain visible through
// this *Values.
func (v *Values) Raw() map[string]any { return v.m }

// requireValue returns the raw value at path or an error if not found.
func (v *Values) requireValue(path string) (any, error) {
	raw := v.Get(path)
	if raw == nil {
		return nil, fmt.Errorf("%w: %q", ErrKeyNotFound, path)
	}
	return raw, nil
}

// String returns the value at path as a string.
func (v *Values) String(path string) (string, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return "", err
	}
	return cast.ToStringE(raw)
}

// StringOr returns the value at path as a string, or def if not found.
func (v *Values) StringOr(path, def string) string {
	raw := v.Get(path)
	if raw == nil {
		return def
	}
	return cast.ToString(raw)
}

// Int returns the value at path as an int.
func (v *Values) Int(path string) (int, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return 0, err
	}
	return cast.ToIntE(raw)
}

// IntOr returns the value at path as an int, or def if not found.
func (v *Values) IntOr(path string, def int) int {
	raw := v.Get(path)
	if raw == nil {
		return def
	}
	return cast.ToInt(raw)
}

// Int64 returns the value at path as an int64.
func (v *Values) Int64(path string) (int64, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return 0, err
	}
	return cast.ToInt64E(raw)
}

// Int64Or returns the value at path as an int64, or def if not found.
func (v *Values) Int64Or(path string, def int64) int64 {
	raw := v.Get(path)
	if raw == nil {
		return def
	}
	return cast.ToInt64(raw)
}

// Float64 returns the value at path as a float64.
func (v *Values) Float64(path string) (float64, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(raw)
}

// Float64Or returns the value at path as a float64, or def if not found.
func (v *Values) Float64Or(path string, def float64) float64 {
	raw := v.Get(path)
	if raw == nil {
		return def
	}
	return cast.ToFloat64(raw)
}

// Bool returns the value at path as a bool.
func (v *Values) Bool(path string) (bool, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return false, err
	}
	return cast.ToBoolE(raw)
}

// BoolOr returns the value at path as a bool, or def if not found.
func (v *Values) BoolOr(path string, def bool) bool {
	raw := v.Get(path)
	if raw == nil {
		return def
	}
	return cast.ToBool(raw)
}

// Duration returns the value at path as a [time.Duration].
func (v *Values) Duration(path string) (time.Duration, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return 0, err
	}
	return cast.ToDurationE(raw)
}

// DurationOr returns the value at path as a [time.Duration], or def if not found.
func (v *Values) DurationOr(path string, def time.Duration) time.Duration {
	raw := v.Get(path)
	if raw == nil {
		return def
	}
	return cast.ToDuration(raw)
}

// Time returns the value at path as a [time.Time].
func (v *Values) Time(path string) (time.Time, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return time.Time{}, err
	}
	return cast.ToTimeE(raw)
}

// StringSlice returns the value at path as a []string.
func (v *Values) StringSlice(path string) ([]string, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringSliceE(raw)
}

// IntSlice returns the value at path as a []int.
func (v *Values) IntSlice(path string) ([]int, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return nil, err
	}
	return cast.ToIntSliceE(raw)
}

// StringMap returns the value at path as a map[string]string.
func (v *Values) StringMap(path string) (map[string]string, error) {
	raw, err := v.requireValue(path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringMapStringE(raw)
}
