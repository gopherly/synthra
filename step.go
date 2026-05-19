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

import "fmt"

// step is the internal interface for a single pipeline stage. Every entry in
// Synthra.steps implements this interface. Steps are executed in the order
// they were registered, after all sources are merged.
type step interface {
	run(values map[string]any) (map[string]any, error)
	// kind returns a short label used in error path strings ("schema",
	// "transform", or "validator").
	kind() string
}

// schemaStep validates values against a JSON Schema and applies the schema's
// declared default values. The schema bytes are resolved lazily via the
// selector each time Load runs, so the selector can inspect the current merged
// values (for example to pick a schema based on an apiVersion field).
type schemaStep struct {
	selector func(map[string]any) ([]byte, error)
}

func (s *schemaStep) kind() string { return "schema" }

func (s *schemaStep) run(values map[string]any) (map[string]any, error) {
	schemaBytes, err := s.selector(values)
	if err != nil {
		return nil, err
	}
	compiled, raw, err := compileJSONSchema(schemaBytes)
	if err != nil {
		return nil, err
	}
	values = canonicalizeSchemaKeys(values, raw)
	values = applySchemaDefaults(values, raw)
	if validationErr := compiled.Validate(values); validationErr != nil {
		return nil, validationErr
	}
	return values, nil
}

// transformStep applies an arbitrary map mutation to the values. The function
// receives the current values and must return the (possibly modified) map.
type transformStep struct {
	fn func(map[string]any) (map[string]any, error)
}

func (s *transformStep) kind() string { return "transform" }

func (s *transformStep) run(values map[string]any) (map[string]any, error) {
	return s.fn(values)
}

// validatorStep runs a read-only check on values without modifying them.
// Panics inside the validator function are recovered and converted into errors.
type validatorStep struct {
	fn func(map[string]any) error
}

func (s *validatorStep) kind() string { return "validator" }

func (s *validatorStep) run(values map[string]any) (result map[string]any, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(error); ok {
				err = fmt.Errorf("validator panic: %w", rerr)
			} else {
				err = fmt.Errorf("validator panic: %v", r)
			}
		}
	}()
	if fnErr := s.fn(values); fnErr != nil {
		return nil, fnErr
	}
	return values, nil
}
