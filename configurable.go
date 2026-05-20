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
	"strings"
)

// Configurable wraps the merged configuration map for pipeline callbacks.
// It embeds [Configuration] for all read operations and adds mutation
// methods (Set, Delete, Walk). Every read method is case-insensitive at
// each path segment. Dot notation addresses nested maps.
//
// Configurable is not safe for concurrent use. Each Load call creates a
// fresh instance for the duration of the pipeline.
//
// Write methods on an element returned by [Configurable.Find] or iterated
// by [Configurable.EachMap] share storage with the parent: mutations reach
// back into the parent's map. Document this at call sites when it matters.
type Configurable struct {
	Configuration
}

// newConfigurable wraps an existing map. The wrapped map is mutated in place.
func newConfigurable(m map[string]any) *Configurable {
	if m == nil {
		m = make(map[string]any)
	}
	return &Configurable{Configuration{m: m}}
}

// Raw returns the underlying map directly (no clone), enabling in-place
// mutation. For a safe read-only copy, call c.Configuration.Raw() or
// use [Configuration.Raw] on the embedded field.
func (c *Configurable) Raw() map[string]any { return c.m }

// Set assigns value at path. Intermediate maps are created as needed.
// Returns an error if a non-final segment along the path exists but is
// not a map.
func (c *Configurable) Set(path string, value any) error {
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		k := findKeyFold(c.m, path)
		if k == "" {
			k = path
		}
		c.m[k] = value
		return nil
	}
	current := c.m
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
func (c *Configurable) Delete(path string) bool {
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		k := findKeyFold(c.m, path)
		if k == "" {
			return false
		}
		delete(c.m, k)
		return true
	}
	current := c.m
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

// Walk visits every node in the tree depth-first. The callback receives the
// dot-path to the node and its current value. Returning (newValue, true)
// replaces the node; returning (_, false) leaves it unchanged. Slice indices
// appear as "key[N]" in the path.
func (c *Configurable) Walk(fn func(path string, value any) (any, bool)) {
	walkMap(c.m, "", fn)
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

// EachMap shadows [Configuration.EachMap] to yield mutable *Configurable wrappers
// instead of read-only *Configuration values. Each wrapper shares the underlying
// map, so Set and Delete on an element mutate the parent tree.
func (c *Configurable) EachMap(path string) iter.Seq2[int, *Configurable] {
	return func(yield func(int, *Configurable) bool) {
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
			if !yield(i, &Configurable{Configuration{m: m}}) {
				return
			}
		}
	}
}

// FindFunc shadows [Configuration.FindFunc] to yield a mutable
// *Configurable wrapper for the matching element. Returns nil if no
// element matches.
func (c *Configurable) FindFunc(path string, pred func(*Configurable) bool) *Configurable {
	for _, entry := range c.EachMap(path) {
		if pred(entry) {
			return entry
		}
	}
	return nil
}

// Find shadows [Configuration.Find] to return a mutable *Configurable
// for the first element at path where field equals match
// (case-insensitive). Returns nil if no element matches.
func (c *Configurable) Find(path, field, match string) *Configurable {
	return c.FindFunc(path, func(e *Configurable) bool {
		return e.StringOr(field, "") == match
	})
}
