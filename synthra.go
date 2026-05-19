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

package synthra

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"sync"

	"github.com/go-viper/mapstructure/v2"
)

// Option is a functional option that can be used to configure an Synthra instance.
// Options apply to an internal config struct; the constructor validates
// and builds the public Synthra from it.
// Options must not be nil; passing nil results in a validation error at
// construction.
type Option func(cfg *config)

// config holds construction-time configuration. Options mutate config;
// New() validates and builds Synthra from it.
type config struct {
	sources          []Source
	dumpers          []Dumper
	binding          any
	bindingHooks     []func(any) error
	tagName          string
	steps            []step
	validationErrors []error
}

// Synthra manages configuration data loaded from multiple sources.
// It provides thread-safe access to configuration values and supports
// binding to structs, validation, and dumping to files.
//
// Synthra is the runtime object returned by New/MustNew; use it for
// Load, Get, and Dump.
// Synthra is safe for concurrent use by multiple goroutines.
type Synthra struct {
	values       *map[string]any
	sources      []Source
	dumpers      []Dumper
	binding      any
	bindingHooks []func(any) error
	tagName      string // Custom struct tag name (default: "synthra")
	steps        []step
	mu           sync.RWMutex
	// decoderConfig holds the cached decoder configuration for struct binding
	decoderConfig *mapstructure.DecoderConfig
	decoderOnce   sync.Once
}

// validate reports any errors collected during option application.
func (cfg *config) validate() error {
	if len(cfg.validationErrors) == 0 {
		return nil
	}
	return errors.Join(cfg.validationErrors...)
}

// defaultConfig returns a config with default values.
func defaultConfig() *config {
	return &config{
		sources: []Source{},
		tagName: "synthra",
	}
}

// configFromConfig builds a Synthra from a validated config.
func configFromConfig(cfg *config) *Synthra {
	return &Synthra{
		values:       &map[string]any{},
		sources:      cfg.sources,
		dumpers:      cfg.dumpers,
		binding:      cfg.binding,
		bindingHooks: cfg.bindingHooks,
		tagName:      cfg.tagName,
		steps:        cfg.steps,
	}
}

// New creates a new [Synthra] instance with the provided options.
// Options are applied in order to an internal config. Validation errors
// are collected and reported after all options are applied, so callers
// never receive a partially-initialized instance. A nil option is
// treated as a validation error.
//
// Use [MustNew] in main() or initialization code where a panic on error
// is acceptable.
//
// Example:
//
//	cfg, err := synthra.New(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvPrefix("APP"),
//	    synthra.WithBinding(&appCfg),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(cfg.Get("server.port"))
func New(opts ...Option) (*Synthra, error) {
	cfg := defaultConfig()
	for i, opt := range opts {
		if opt == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, fmt.Sprintf("option[%d]", i), errors.New("cannot be nil")))
			continue
		}
		opt(cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return configFromConfig(cfg), nil
}

// MustNew is like [New] but panics if validation fails.
// Use it in main() or package-level initialization where a panic is
// acceptable. For explicit error handling, use [New] instead.
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvPrefix("APP"),
//	    synthra.WithBinding(&appCfg),
//	)
//	fmt.Println(cfg.Get("server.port"))
func MustNew(opts ...Option) *Synthra {
	cfg, err := New(opts...)
	if err != nil {
		panic(fmt.Sprintf("synthra: validation failed: %v", err))
	}
	return cfg
}

// deepMerge merges src into dst recursively. When both dst and src have a map
// value for the same key (matched case-insensitively), the maps are merged
// recursively and the first writer's key casing is kept. For all other value
// types the src value overrides the dst value under the existing dst key.
func deepMerge(dst, src map[string]any) {
	for srcKey, srcVal := range src {
		dstKey := findKeyFold(dst, srcKey)
		if dstKey == "" {
			if srcMap, ok := srcVal.(map[string]any); ok {
				nested := make(map[string]any)
				deepMerge(nested, srcMap)
				dst[srcKey] = nested
			} else {
				dst[srcKey] = srcVal
			}
			continue
		}
		if srcMap, srcIsMap := srcVal.(map[string]any); srcIsMap {
			if dstMap, dstIsMap := dst[dstKey].(map[string]any); dstIsMap {
				deepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[dstKey] = srcVal
	}
}

// loadSourcesSequential loads configuration data from all sources
// sequentially to avoid race conditions.
func (c *Synthra) loadSourcesSequential(ctx context.Context) (map[string]any, error) {
	if len(c.sources) == 0 {
		return make(map[string]any), nil
	}

	// Merge to maintain precedence
	newValues := make(map[string]any)
	for i, src := range c.sources {
		if ctx.Err() != nil {
			return nil, NewConfigError(OpLoad, fmt.Sprintf("source[%d]", i), ctx.Err())
		}

		conf, err := src.Load(ctx)
		if err != nil {
			return nil, NewConfigError(OpLoad, fmt.Sprintf("source[%d]", i), err)
		}

		// Ensure we always have a valid map, even if source returns nil
		if conf == nil {
			conf = make(map[string]any)
		}

		deepMerge(newValues, conf)
	}

	return newValues, nil
}

// Load loads configuration data from the registered sources and merges it
// into the internal values map. The method executes all registered pipeline
// steps in registration order before atomically updating the internal state.
// Load is safe to call concurrently.
//
// The pipeline runs in this order:
//  1. Load and merge all sources (later sources override earlier ones).
//  2. Execute each registered step ([WithJSONSchema], [WithJSONSchemaFunc],
//     [WithTransform], [WithEnvSubst], [WithValidator]) in the order they
//     were registered.
//  3. Decode into the bound struct ([WithBinding]), apply struct-tag defaults,
//     and call the struct's Validate method if it implements [Validator].
//
// Errors:
//   - Returns [*ConfigError] with [OpLoad] if ctx is nil ([ErrNilContext])
//   - Returns [*ConfigError] with [OpLoad] if any source fails to load or merge
//   - Returns [*ConfigError] with [OpLoad] and Path "step[N]:schema" if a schema
//     step's selector, compilation, or validation fails
//   - Returns [*ConfigError] with [OpLoad] and Path "step[N]:transform" if a
//     transform step returns an error
//   - Returns [*ConfigError] with [OpLoad] and Path "step[N]:validator" if a
//     validator step returns an error or panics
//   - Returns [*ConfigError] with [OpLoad] if binding or struct validation fails
func (c *Synthra) Load(ctx context.Context) error {
	if ctx == nil {
		return NewConfigError(OpLoad, "", ErrNilContext)
	}

	newValues, err := c.loadSourcesSequential(ctx)
	if err != nil {
		return err
	}

	for i, s := range c.steps {
		newValues, err = s.run(newValues)
		if err != nil {
			return NewConfigError(OpLoad, fmt.Sprintf("step[%d]:%s", i, s.kind()), err)
		}
		if newValues == nil {
			newValues = make(map[string]any)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.binding != nil {
		bindingType := reflect.TypeOf(c.binding)
		if bindingType.Kind() == reflect.Pointer {
			bindingType = bindingType.Elem()
		}
		tempBinding := reflect.New(bindingType).Interface()

		if bindErr := c.decodeBindingInto(tempBinding, &newValues); bindErr != nil {
			return NewConfigError(OpLoad, "binding-decode", bindErr)
		}
		if bindErr := applyDefaults(tempBinding); bindErr != nil {
			return NewConfigError(OpLoad, "binding-defaults", bindErr)
		}
		for i, hook := range c.bindingHooks {
			if hookErr := hook(tempBinding); hookErr != nil {
				return NewConfigError(OpLoad, fmt.Sprintf("binding-hook[%d]", i), hookErr)
			}
		}
		if v, ok := tempBinding.(Validator); ok {
			if validateErr := v.Validate(); validateErr != nil {
				return NewConfigError(OpLoad, "binding-validate", validateErr)
			}
		}

		// tempBinding decoded and validated successfully; copy it into the
		// real binding. A second decode would succeed identically, so we
		// avoid the redundant work by copying via reflection.
		reflect.ValueOf(c.binding).Elem().Set(reflect.ValueOf(tempBinding).Elem())
	}

	c.values = &newValues

	return nil
}

// Dump writes the current configuration values to the registered dumpers.
//
// Errors:
//   - Returns [*ConfigError] with [OpDump] if ctx is nil ([ErrNilContext])
//   - Returns [*ConfigError] with [OpDump] if any dumper fails to write the
//     configuration
func (c *Synthra) Dump(ctx context.Context) error {
	if ctx == nil {
		return NewConfigError(OpDump, "", ErrNilContext)
	}

	// Get a copy of the values to avoid holding locks during dumper calls
	var valuesCopy map[string]any
	func() {
		c.mu.RLock()
		defer c.mu.RUnlock()
		if c.values != nil {
			// Use shallow copy for better performance
			valuesCopy = make(map[string]any, len(*c.values))
			maps.Copy(valuesCopy, *c.values)
		} else {
			valuesCopy = make(map[string]any)
		}
	}()

	for i, d := range c.dumpers {
		if err := d.Dump(ctx, &valuesCopy); err != nil {
			return NewConfigError(OpDump, fmt.Sprintf("dumper[%d]", i), err)
		}
	}

	return nil
}

// Values returns a pointer to a shallow copy of the loaded configuration map.
// The copy is taken while holding a read lock; nested maps, slices, and
// pointers inside values are not deep-copied, so mutating nested data still
// affects the same objects held by this Synthra.
// If Load has not run yet, it returns a pointer to a new empty map.
func (c *Synthra) Values() *map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.values == nil {
		m := make(map[string]any)
		return &m
	}

	cloned := maps.Clone(*c.values)
	return &cloned
}

// getValueFromMap retrieves the value associated with the given path from
// the internal values map. The path is a dot-separated string that
// represents the nested structure of the map. Lookup is case-insensitive
// at each segment using findKeyFold.
func (c *Synthra) getValueFromMap(path string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.values == nil {
		return nil
	}

	current := *c.values

	// 1. Check for a direct (top-level) key match first. This handles the
	// uncommon case where a key itself contains a literal dot.
	if k := findKeyFold(current, path); k != "" {
		return current[k]
	}

	// 2. Fallback to dot-notation traversal. Navigate every segment except
	// the last into the nested maps, then return the final segment's value.
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		return nil // already checked above
	}
	for _, segment := range segments[:len(segments)-1] {
		k := findKeyFold(current, segment)
		if k == "" {
			return nil
		}
		nested, isMap := current[k].(map[string]any)
		if !isMap {
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

// requireValue returns the raw value at key for strict typed accessors and [Get].
// It returns [ErrNilConfig] if c is nil, and an error wrapping [ErrKeyNotFound]
// if the key is empty or not present.
func (c *Synthra) requireValue(key string) (any, error) {
	if c == nil {
		return nil, ErrNilConfig
	}
	if key == "" {
		return nil, fmt.Errorf("%w: empty key", ErrKeyNotFound)
	}
	v := c.getValueFromMap(key)
	if v == nil {
		return nil, fmt.Errorf("%w: %q", ErrKeyNotFound, key)
	}
	return v, nil
}

// Get returns the value associated with the given key as an any type.
// If the key is not found, it returns nil.
func (c *Synthra) Get(key string) any {
	if c == nil {
		return nil
	}
	if key == "" {
		return nil
	}
	return c.getValueFromMap(key)
}
