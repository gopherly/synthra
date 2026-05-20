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
	"iter"
	"maps"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// Configuration is the public, immutable read view of the merged configuration.
// It is returned by [Synthra.Configuration] and also embedded inside the
// mutable [Configurable] that pipeline callbacks receive.
//
// All accessor methods are case-insensitive and use dot-notation for nested
// paths. Every *Or method is nil-safe: calling it on a nil *Configuration
// returns the supplied default rather than panicking. Strict (non-Or) accessors
// return [ErrNilConfig] on a nil receiver.
type Configuration struct {
	m map[string]any
}

func emptyConfiguration() *Configuration {
	return &Configuration{m: map[string]any{}}
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

// Get returns the raw value at path or nil if no key matches.
func (c *Configuration) Get(path string) any {
	if c == nil {
		return nil
	}
	return lookupPath(c.m, path)
}

// Has reports whether path exists in the configuration.
func (c *Configuration) Has(path string) bool {
	if c == nil {
		return false
	}
	return hasPath(c.m, path)
}

// Keys returns the top-level keys in their stored casing.
func (c *Configuration) Keys() []string {
	if c == nil {
		return nil
	}
	keys := make([]string, 0, len(c.m))
	for k := range c.m {
		keys = append(keys, k)
	}
	return keys
}

// Raw returns a shallow copy of the underlying map.
func (c *Configuration) Raw() map[string]any {
	if c == nil {
		return nil
	}
	return maps.Clone(c.m)
}

// String returns the value at path as a string.
func (c *Configuration) String(path string) (string, error) {
	if c == nil {
		return "", NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return "", err
	}
	return cast.ToStringE(raw)
}

// StringOr returns the value at path as a string, or def if not found
// or nil receiver.
func (c *Configuration) StringOr(path, def string) string {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToString(raw)
}

// Int returns the value at path as an int.
func (c *Configuration) Int(path string) (int, error) {
	if c == nil {
		return 0, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToIntE(raw)
}

// IntOr returns the value at path as an int, or def if not found or nil receiver.
func (c *Configuration) IntOr(path string, def int) int {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToInt(raw)
}

// Int64 returns the value at path as an int64.
func (c *Configuration) Int64(path string) (int64, error) {
	if c == nil {
		return 0, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToInt64E(raw)
}

// Int64Or returns the value at path as an int64, or def if not found
// or nil receiver.
func (c *Configuration) Int64Or(path string, def int64) int64 {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToInt64(raw)
}

// Float64 returns the value at path as a float64.
func (c *Configuration) Float64(path string) (float64, error) {
	if c == nil {
		return 0, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToFloat64E(raw)
}

// Float64Or returns the value at path as a float64, or def if not
// found or nil receiver.
func (c *Configuration) Float64Or(path string, def float64) float64 {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToFloat64(raw)
}

// Bool returns the value at path as a bool.
func (c *Configuration) Bool(path string) (bool, error) {
	if c == nil {
		return false, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return false, err
	}
	return cast.ToBoolE(raw)
}

// BoolOr returns the value at path as a bool, or def if not found or nil receiver.
func (c *Configuration) BoolOr(path string, def bool) bool {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToBool(raw)
}

// Duration returns the value at path as a [time.Duration].
func (c *Configuration) Duration(path string) (time.Duration, error) {
	if c == nil {
		return 0, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return 0, err
	}
	return cast.ToDurationE(raw)
}

// DurationOr returns the value at path as a [time.Duration], or def
// if not found or nil receiver.
func (c *Configuration) DurationOr(path string, def time.Duration) time.Duration {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToDuration(raw)
}

// Time returns the value at path as a [time.Time].
func (c *Configuration) Time(path string) (time.Time, error) {
	if c == nil {
		return time.Time{}, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return time.Time{}, err
	}
	return cast.ToTimeE(raw)
}

// TimeOr returns the value at path as a [time.Time], or def if not
// found or nil receiver.
func (c *Configuration) TimeOr(path string, def time.Time) time.Time {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToTime(raw)
}

// StringSlice returns the value at path as a []string.
func (c *Configuration) StringSlice(path string) ([]string, error) {
	if c == nil {
		return nil, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringSliceE(raw)
}

// StringSliceOr returns the value at path as a []string, or def if
// not found or nil receiver.
func (c *Configuration) StringSliceOr(path string, def []string) []string {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToStringSlice(raw)
}

// IntSlice returns the value at path as a []int.
func (c *Configuration) IntSlice(path string) ([]int, error) {
	if c == nil {
		return nil, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToIntSliceE(raw)
}

// IntSliceOr returns the value at path as a []int, or def if not
// found or nil receiver.
func (c *Configuration) IntSliceOr(path string, def []int) []int {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToIntSlice(raw)
}

// StringMap returns the value at path as a map[string]any.
func (c *Configuration) StringMap(path string) (map[string]any, error) {
	if c == nil {
		return nil, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringMapE(raw)
}

// StringMapOr returns the value at path as a map[string]any, or def
// if not found or nil receiver.
func (c *Configuration) StringMapOr(path string, def map[string]any) map[string]any {
	if c == nil {
		return def
	}
	raw := lookupPath(c.m, path)
	if raw == nil {
		return def
	}
	return cast.ToStringMap(raw)
}

// StringMapString returns the value at path as a map[string]string.
func (c *Configuration) StringMapString(path string) (map[string]string, error) {
	if c == nil {
		return nil, NewConfigError(OpGet, path, ErrNilConfig)
	}
	raw, err := requirePath(c.m, path)
	if err != nil {
		return nil, err
	}
	return cast.ToStringMapStringE(raw)
}

// SliceLen returns the length of the slice at path.
// Returns 0 if the path is missing, the value is not a slice, or the
// receiver is nil.
func (c *Configuration) SliceLen(path string) int {
	if c == nil {
		return 0
	}
	raw := lookupPath(c.m, path)
	if slice, ok := raw.([]any); ok {
		return len(slice)
	}
	return 0
}

// EachMap returns an iterator over map elements at path. Non-map elements in
// the slice are skipped. Missing path or non-slice values yield nothing. The
// returned [Configuration] shares the underlying map: mutations through a
// [Configurable] wrapper (see [Configurable.EachMap]) reach back to the parent.
func (c *Configuration) EachMap(path string) iter.Seq2[int, *Configuration] {
	return func(yield func(int, *Configuration) bool) {
		if c == nil {
			return
		}
		slice, ok := lookupPath(c.m, path).([]any)
		if !ok {
			return
		}
		for i, elem := range slice {
			m, isMap := elem.(map[string]any)
			if !isMap {
				continue
			}
			if !yield(i, &Configuration{m: m}) {
				return
			}
		}
	}
}

// FindFunc returns the first map element at path for which pred returns true,
// or nil if no element matches. Non-map elements in the slice are skipped.
// Because the returned *Configuration shares the underlying map, nil-safe
// *Or methods on it compose cleanly without guards.
func (c *Configuration) FindFunc(path string, pred func(*Configuration) bool) *Configuration {
	for _, entry := range c.EachMap(path) {
		if pred(entry) {
			return entry
		}
	}
	return nil
}

// Find returns the first map element at path where field equals match
// (case-insensitive). Returns nil if no element matches.
func (c *Configuration) Find(path, field, match string) *Configuration {
	return c.FindFunc(path, func(e *Configuration) bool {
		return e.StringOr(field, "") == match
	})
}
