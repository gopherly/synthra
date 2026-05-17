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

	"dario.cat/mergo"
	"github.com/go-viper/mapstructure/v2"
	"github.com/santhosh-tekuri/jsonschema/v6"
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
	sources            []Source
	dumpers            []Dumper
	binding            any
	tagName            string
	jsonSchemaCompiled *jsonschema.Schema
	jsonSchemaRaw      map[string]any // raw parsed schema for default extraction
	jsonSchemaSelector func(map[string]any) ([]byte, error)
	customValidators   []func(map[string]any) error
	transforms         []func(map[string]any) (map[string]any, error)
	validationErrors   []error
}

// Synthra manages configuration data loaded from multiple sources.
// It provides thread-safe access to configuration values and supports
// binding to structs, validation, and dumping to files.
//
// Synthra is the runtime object returned by New/MustNew; use it for
// Load, Get, and Dump.
// Synthra is safe for concurrent use by multiple goroutines.
type Synthra struct {
	values             *map[string]any
	sources            []Source
	dumpers            []Dumper
	binding            any
	tagName            string // Custom struct tag name (default: "synthra")
	mu                 sync.RWMutex
	jsonSchemaCompiled *jsonschema.Schema
	jsonSchemaRaw      map[string]any // raw parsed schema for default extraction
	jsonSchemaSelector func(map[string]any) ([]byte, error)
	customValidators   []func(map[string]any) error
	transforms         []func(map[string]any) (map[string]any, error)
	// decoderConfig holds the cached decoder configuration for struct binding
	decoderConfig *mapstructure.DecoderConfig
	decoderOnce   sync.Once
}

// validate reports any errors collected during option application.
func (cfg *config) validate() error {
	if cfg.jsonSchemaCompiled != nil && cfg.jsonSchemaSelector != nil {
		cfg.validationErrors = append(cfg.validationErrors,
			NewConfigError(OpNew, "WithJSONSchemaSelector",
				errors.New("WithJSONSchema and WithJSONSchemaSelector are mutually exclusive")))
	}
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
		values:             &map[string]any{},
		sources:            cfg.sources,
		dumpers:            cfg.dumpers,
		binding:            cfg.binding,
		tagName:            cfg.tagName,
		jsonSchemaCompiled: cfg.jsonSchemaCompiled,
		jsonSchemaRaw:      cfg.jsonSchemaRaw,
		jsonSchemaSelector: cfg.jsonSchemaSelector,
		customValidators:   cfg.customValidators,
		transforms:         cfg.transforms,
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

// normalizeMapKeys recursively converts all map keys to lowercase for
// case-insensitive merging
func normalizeMapKeys(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	normalized := make(map[string]any)
	for k, v := range m {
		lowerKey := strings.ToLower(k)
		if nestedMap, ok := v.(map[string]any); ok {
			normalized[lowerKey] = normalizeMapKeys(nestedMap)
		} else {
			normalized[lowerKey] = v
		}
	}
	return normalized
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

		// Normalize keys to lowercase for case-insensitive merging
		normalizedConf := normalizeMapKeys(conf)

		// Use mergo to merge configuration maps with override behavior
		if err = mergo.Map(&newValues, normalizedConf, mergo.WithOverride); err != nil {
			return nil, NewConfigError(OpLoad, fmt.Sprintf("source[%d]", i), err)
		}
	}

	return newValues, nil
}

// Load loads configuration data from the registered sources and merges it
// into the internal values map. The method validates the configuration data
// before atomically updating the internal state.
// Load is safe to call concurrently.
//
// The pipeline runs in this order:
//  1. Load and merge all sources (later sources override earlier ones).
//  2. If [WithJSONSchemaSelector] is set, call it with the merged values to
//     obtain schema bytes; compile them and extract raw defaults.
//  3. Apply JSON Schema defaults ([WithJSONSchema] or selector) to missing keys.
//  4. Run transforms ([WithTransform], [WithInterpolation]) in registration order.
//  5. Validate merged values against the JSON Schema.
//  6. Run custom validators ([WithValidator]).
//  7. Decode into the bound struct ([WithBinding]), apply struct-tag defaults,
//     and call the struct's Validate method if it implements [Validator].
//
// Errors:
//   - Returns [*ConfigError] with [OpLoad] if ctx is nil ([ErrNilContext])
//   - Returns [*ConfigError] with [OpLoad] if any source fails to load or merge
//   - Returns [*ConfigError] with [OpLoad] and Path "json-schema-selector" if the
//     selector returns an error or the returned bytes fail to compile
//   - Returns [*ConfigError] with [OpLoad] and Path "transform[N]" if a transform
//     returns an error
//   - Returns [*ConfigError] with [OpLoad] and Path "json-schema" if JSON schema
//     validation fails
//   - Returns [*ConfigError] with [OpLoad] if custom validators fail
//   - Returns [*ConfigError] with [OpLoad] if binding or struct validation fails
func (c *Synthra) Load(ctx context.Context) error {
	if ctx == nil {
		return NewConfigError(OpLoad, "", ErrNilContext)
	}

	newValues, err := c.loadSourcesSequential(ctx)
	if err != nil {
		return err
	}

	// Ensure newValues is never nil
	if newValues == nil {
		newValues = make(map[string]any)
	}

	// Resolve the schema for this load. For static schemas (WithJSONSchema)
	// these are already set on c. For selector schemas (WithJSONSchemaSelector)
	// we compute them per-load using local variables so concurrent Load calls
	// never race on the shared Synthra fields.
	loadSchemaCompiled := c.jsonSchemaCompiled
	loadSchemaRaw := c.jsonSchemaRaw

	if c.jsonSchemaSelector != nil {
		schemaBytes, selErr := c.jsonSchemaSelector(newValues)
		if selErr != nil {
			return NewConfigError(OpLoad, "json-schema-selector", selErr)
		}
		compiled, raw, compErr := compileJSONSchema(schemaBytes)
		if compErr != nil {
			return NewConfigError(OpLoad, "json-schema-selector", compErr)
		}
		loadSchemaCompiled = compiled
		loadSchemaRaw = raw
	}

	// Apply JSON Schema defaults to any keys that are missing from the
	// loaded config. This runs before transforms and validation so that
	// transforms and the schema validator see a fully populated map.
	if loadSchemaRaw != nil {
		newValues = applySchemaDefaults(newValues, loadSchemaRaw)
	}

	// Run registered transforms in order. Transforms receive the current
	// values (with schema defaults already applied) and may return a
	// modified map. They run before validation so the validated values
	// are the final, transformed values.
	for i, fn := range c.transforms {
		newValues, err = fn(newValues)
		if err != nil {
			return NewConfigError(OpLoad, fmt.Sprintf("transform[%d]", i), err)
		}
		if newValues == nil {
			newValues = make(map[string]any)
		}
	}

	if loadSchemaCompiled != nil {
		if err = loadSchemaCompiled.Validate(newValues); err != nil {
			return NewConfigError(OpLoad, "json-schema", err)
		}
	}

	// Custom function validators
	for i, fn := range c.customValidators {
		var validatorErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					if rerr, ok := r.(error); ok {
						validatorErr = fmt.Errorf("validator panic: %w", rerr)
					} else {
						validatorErr = fmt.Errorf("validator panic: %v", r)
					}
				}
			}()
			validatorErr = fn(newValues)
		}()
		if validatorErr != nil {
			return NewConfigError(OpLoad, fmt.Sprintf("custom-validator[%d]", i), validatorErr)
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
		if v, ok := tempBinding.(Validator); ok {
			if validateErr := v.Validate(); validateErr != nil {
				return NewConfigError(OpLoad, "binding-validate", validateErr)
			}
		}

		if bindErr := c.decodeBindingInto(c.binding, &newValues); bindErr != nil {
			return NewConfigError(OpLoad, "binding-decode", bindErr)
		}
		if bindErr := applyDefaults(c.binding); bindErr != nil {
			return NewConfigError(OpLoad, "binding-defaults", bindErr)
		}
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
// represents the nested structure of the map. If the path is valid and
// the final value is found, it is returned. Otherwise, nil is returned.
// Keys are case-insensitive since they are stored in lowercase.
func (c *Synthra) getValueFromMap(path string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.values == nil {
		return nil
	}

	// Work with a copy of the current map to avoid race conditions during traversal
	current := *c.values

	// Normalize the path to lowercase for case-insensitive lookup
	normalizedPath := strings.ToLower(path)

	// 1. Check for direct key match first
	if val, ok := current[normalizedPath]; ok {
		return val
	}

	// 2. Fallback to dot notation traversal
	segments := strings.Split(normalizedPath, ".")
	for i, segment := range segments {
		if currentMap, ok := current[segment]; ok {
			if i == len(segments)-1 {
				return currentMap
			}
			if nestedMap, isMap := currentMap.(map[string]any); isMap {
				current = nestedMap
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return nil
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
