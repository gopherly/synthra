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
	"maps"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// Reader is the read-only contract satisfied by both [*Snapshot] and [*Values].
// Use Reader in function signatures (such as validators) where the caller
// must not mutate configuration state.
type Reader interface {
	Get(path string) any
	Has(path string) bool
	Keys() []string
	String(path string) (string, error)
	StringOr(path, def string) string
	Int(path string) (int, error)
	IntOr(path string, def int) int
	Int64(path string) (int64, error)
	Int64Or(path string, def int64) int64
	Float64(path string) (float64, error)
	Float64Or(path string, def float64) float64
	Bool(path string) (bool, error)
	BoolOr(path string, def bool) bool
	Duration(path string) (time.Duration, error)
	DurationOr(path string, def time.Duration) time.Duration
	Time(path string) (time.Time, error)
	TimeOr(path string, def time.Time) time.Time
	StringSlice(path string) ([]string, error)
	StringSliceOr(path string, def []string) []string
	IntSlice(path string) ([]int, error)
	IntSliceOr(path string, def []int) []int
	StringMap(path string) (map[string]any, error)
	StringMapOr(path string, def map[string]any) map[string]any
	StringMapString(path string) (map[string]string, error)
}

// Compile-time interface satisfaction checks.
var (
	_ Reader = (*Snapshot)(nil)
	_ Reader = (*Values)(nil)
)

// Snapshot is the public, immutable read view returned by [Synthra.Snapshot].
// It holds a point-in-time copy of the loaded configuration. All typed accessor
// methods are defined on Snapshot; after Load completes, readers can call
// cfg.Snapshot().String("key") without holding any lock.
type Snapshot struct {
	m map[string]any
}

func emptySnapshot() *Snapshot {
	return &Snapshot{m: map[string]any{}}
}

// lookupPath resolves a dot-separated path in a map using case-insensitive
// matching at each segment. A top-level key containing a literal dot is
// checked first before falling back to dot-notation traversal.
func lookupPath(m map[string]any, path string) any {
	if k := findKeyFold(m, path); k != "" {
		return m[k]
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return nil
	}
	current := m
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

// hasPath reports whether a dot-separated path exists in the map.
func hasPath(m map[string]any, path string) bool {
	if findKeyFold(m, path) != "" {
		return true
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return false
	}
	current := m
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

// requirePath returns the raw value at path or an error wrapping ErrKeyNotFound.
func requirePath(m map[string]any, path string) (any, error) {
	raw := lookupPath(m, path)
	if raw == nil {
		return nil, fmt.Errorf("%w: %q", ErrKeyNotFound, path)
	}
	return raw, nil
}

// --- Snapshot accessors (read-only) ---

// Get returns the raw value at path or nil if no key matches.
func (s *Snapshot) Get(path string) any { return lookupPath(s.m, path) }

// Has reports whether path exists.
func (s *Snapshot) Has(path string) bool { return hasPath(s.m, path) }

// Keys returns the top-level keys in their stored casing.
func (s *Snapshot) Keys() []string {
	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}
	return keys
}

// Raw returns a shallow copy of the underlying map.
func (s *Snapshot) Raw() map[string]any { return maps.Clone(s.m) }

// String returns the value at path as a string.
func (s *Snapshot) String(path string) (string, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return "", err
	}
	return cast.ToStringE(raw)
}

// StringOr returns the value at path as a string, or def if not found.
func (s *Snapshot) StringOr(path, def string) string {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToString(raw)
}

// Int returns the value at path as an int.
func (s *Snapshot) Int(path string) (int, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToIntE(raw)
}

// IntOr returns the value at path as an int, or def if not found.
func (s *Snapshot) IntOr(path string, def int) int {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToInt(raw)
}

// Int64 returns the value at path as an int64.
func (s *Snapshot) Int64(path string) (int64, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToInt64E(raw)
}

// Int64Or returns the value at path as an int64, or def if not found.
func (s *Snapshot) Int64Or(path string, def int64) int64 {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToInt64(raw)
}

// Float64 returns the value at path as a float64.
func (s *Snapshot) Float64(path string) (float64, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(raw)
}

// Float64Or returns the value at path as a float64, or def if not found.
func (s *Snapshot) Float64Or(path string, def float64) float64 {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToFloat64(raw)
}

// Bool returns the value at path as a bool.
func (s *Snapshot) Bool(path string) (bool, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return false, err
	}
	return cast.ToBoolE(raw)
}

// BoolOr returns the value at path as a bool, or def if not found.
func (s *Snapshot) BoolOr(path string, def bool) bool {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToBool(raw)
}

// Duration returns the value at path as a [time.Duration].
func (s *Snapshot) Duration(path string) (time.Duration, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToDurationE(raw)
}

// DurationOr returns the value at path as a [time.Duration], or def if not found.
func (s *Snapshot) DurationOr(path string, def time.Duration) time.Duration {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToDuration(raw)
}

// Time returns the value at path as a [time.Time].
func (s *Snapshot) Time(path string) (time.Time, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return time.Time{}, err
	}
	return cast.ToTimeE(raw)
}

// TimeOr returns the value at path as a [time.Time], or def if not found.
func (s *Snapshot) TimeOr(path string, def time.Time) time.Time {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToTime(raw)
}

// StringSlice returns the value at path as a []string.
func (s *Snapshot) StringSlice(path string) ([]string, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringSliceE(raw)
}

// StringSliceOr returns the value at path as a []string, or def if not found.
func (s *Snapshot) StringSliceOr(path string, def []string) []string {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToStringSlice(raw)
}

// IntSlice returns the value at path as a []int.
func (s *Snapshot) IntSlice(path string) ([]int, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToIntSliceE(raw)
}

// IntSliceOr returns the value at path as a []int, or def if not found.
func (s *Snapshot) IntSliceOr(path string, def []int) []int {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToIntSlice(raw)
}

// StringMap returns the value at path as a map[string]any.
func (s *Snapshot) StringMap(path string) (map[string]any, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringMapE(raw)
}

// StringMapOr returns the value at path as a map[string]any, or def if not found.
func (s *Snapshot) StringMapOr(path string, def map[string]any) map[string]any {
	raw := lookupPath(s.m, path)
	if raw == nil {
		return def
	}
	return cast.ToStringMap(raw)
}

// StringMapString returns the value at path as a map[string]string.
func (s *Snapshot) StringMapString(path string) (map[string]string, error) {
	raw, err := requirePath(s.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringMapStringE(raw)
}
