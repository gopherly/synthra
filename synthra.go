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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/go-viper/mapstructure/v2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cast"
	"gopherly.dev/synthra/codec"
	"gopherly.dev/synthra/dumper"
	"gopherly.dev/synthra/source"
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
	customValidators   []func(map[string]any) error
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
	customValidators   []func(map[string]any) error
	// decoderConfig holds the cached decoder configuration for struct binding
	decoderConfig *mapstructure.DecoderConfig
	decoderOnce   sync.Once
}

// WithSource adds a source to the configuration loader.
func WithSource(loader Source) Option {
	return func(cfg *config) {
		if loader == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithSource", errors.New("source cannot be nil")))
			return
		}
		cfg.sources = append(cfg.sources, loader)
	}
}

// WithFileDumper returns an Option that configures the Synthra instance
// to dump configuration data to a file.
// The format is automatically detected from the file extension (.yaml,
// .yml, .json, .toml).
// For files without extensions or custom formats, use WithFileDumperAs instead.
//
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
// Example: "${LOG_DIR}/config.yaml" expands to "/var/log/config.yaml"
// when LOG_DIR=/var/log
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithFileDumper("output.yaml"),  // Auto-detects YAML
//	)
func WithFileDumper(path string) Option {
	return func(cfg *config) {
		path = os.ExpandEnv(path)

		c, err := detectFormat(path)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithFileDumper", err))
			return
		}

		cfg.dumpers = append(cfg.dumpers, dumper.NewFile(path, c))
	}
}

// WithDumper adds a dumper to the configuration loader.
func WithDumper(d Dumper) Option {
	return func(cfg *config) {
		if d == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithDumper", errors.New("dumper cannot be nil")))
			return
		}
		cfg.dumpers = append(cfg.dumpers, d)
	}
}

// WithFile returns an Option that configures the Synthra instance to
// load configuration data from a file.
// The format is automatically detected from the file extension (.yaml,
// .yml, .json, .toml).
// For files without extensions or custom formats, use WithFileAs instead.
//
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
// Example: "${CONFIG_DIR}/app.yaml" expands to "/etc/myapp/app.yaml"
// when CONFIG_DIR=/etc/myapp
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),     // Automatically detects YAML
//	    synthra.WithFile("override.json"),   // Automatically detects JSON
//	)
func WithFile(path string) Option {
	return func(cfg *config) {
		path = os.ExpandEnv(path)

		c, err := detectFormat(path)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithFile", err))
			return
		}

		cfg.sources = append(cfg.sources, source.NewFile(path, c))
	}
}

// WithFileFS returns an Option that loads configuration from path inside fsys.
// The format is detected from path's file extension, like [WithFile].
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
//
// If fsys is nil, New returns a validation error at construction.
//
// Example (tests with [testing/fstest.MapFS]):
//
//	fsys := fstest.MapFS{"app.yaml": &fstest.MapFile{Data: []byte("port: 8080\n")}}
//	cfg := synthra.MustNew(synthra.WithFileFS(fsys, "app.yaml"))
func WithFileFS(fsys fs.FS, path string) Option {
	return func(cfg *config) {
		if fsys == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithFileFS", errors.New("filesystem cannot be nil")))
			return
		}

		path = os.ExpandEnv(path)

		c, err := detectFormat(path)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithFileFS", err))
			return
		}

		cfg.sources = append(cfg.sources, source.NewFileFS(fsys, path, c))
	}
}

// WithFileFSAs returns an Option that loads configuration from path inside fsys
// using an explicit decoder, like [WithFileAs].
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
//
// If fsys is nil, New returns a validation error at construction.
func WithFileFSAs(fsys fs.FS, path string, decoder codec.Decoder) Option {
	return func(cfg *config) {
		if fsys == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithFileFSAs", errors.New("filesystem cannot be nil")))
			return
		}

		path = os.ExpandEnv(path)
		cfg.sources = append(cfg.sources, source.NewFileFS(fsys, path, decoder))
	}
}

// WithEnv returns an Option that configures the Synthra instance to load
// configuration data from environment variables.
// The prefix parameter specifies the prefix for the environment variables
// to be loaded.
// Environment variables are converted to lowercase and underscores create
// nested structures.
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnv("APP_"),  // Loads APP_SERVER_PORT as server.port
//	)
func WithEnv(prefix string) Option {
	return func(cfg *config) {
		cfg.sources = append(cfg.sources, source.NewOSEnvVar(prefix))
	}
}

// WithConsul returns an Option that configures the Synthra instance to
// load configuration data from a Consul server.
// The format is automatically detected from the path extension.
// For custom formats, use WithConsulAs instead.
//
// CONSUL_HTTP_ADDR is required. If it is not set, New/MustNew returns a
// validation error at construction.
// For optional Consul (e.g., development without Consul), use
// WithConsulOptional instead.
//
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
// Example: "${APP_ENV}/service.yaml" expands to "production/service.yaml"
// when APP_ENV=production
//
// Required environment variables:
//   - CONSUL_HTTP_ADDR: The address of the Consul server
//     (e.g., "http://localhost:8500")
//   - CONSUL_HTTP_TOKEN: The access token for authentication with Consul
//     (optional)
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithConsul("production/service.yaml"),  // Fails at construction if CONSUL_HTTP_ADDR is unset
//	)
func WithConsul(path string) Option {
	return func(cfg *config) {
		if os.Getenv("CONSUL_HTTP_ADDR") == "" {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsul", errors.New("CONSUL_HTTP_ADDR is not set")))
			return
		}

		path = os.ExpandEnv(path)

		c, err := detectFormat(path)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsul", err))
			return
		}

		l, err := source.NewConsul(path, c, nil)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsul", err))
			return
		}

		cfg.sources = append(cfg.sources, l)
	}
}

// WithFileAs returns an Option that configures the Synthra instance to
// load configuration data from a file with explicit decoder.
// Use this when the file doesn't have an extension or when you need to
// override the format detection.
//
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
// Example: "${CONFIG_DIR}/app" expands to "/etc/myapp/app" when
// CONFIG_DIR=/etc/myapp
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFileAs("config", codec.YAML()),      // No extension, specify YAML
//	    synthra.WithFileAs("config.dat", codec.JSON()),  // Wrong extension, specify JSON
//	)
func WithFileAs(path string, decoder codec.Decoder) Option {
	return func(cfg *config) {
		path = os.ExpandEnv(path)
		cfg.sources = append(cfg.sources, source.NewFile(path, decoder))
	}
}

// WithConsulAs returns an Option that configures the Synthra instance to
// load configuration data from a Consul server with explicit decoder.
// Use this when you need to override the format detection.
//
// CONSUL_HTTP_ADDR is required. If it is not set, New/MustNew returns a
// validation error at construction.
// For optional Consul (e.g., development without Consul), use
// WithConsulAsOptional instead.
//
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
// Example: "${APP_ENV}/service" expands to "production/service" when
// APP_ENV=production
//
// Required environment variables:
//   - CONSUL_HTTP_ADDR: The address of the Consul server
//     (e.g., "http://localhost:8500")
//   - CONSUL_HTTP_TOKEN: The access token for authentication with Consul
//     (optional)
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithConsulAs("production/service", codec.JSON()),
//	)
func WithConsulAs(path string, decoder codec.Decoder) Option {
	return func(cfg *config) {
		if os.Getenv("CONSUL_HTTP_ADDR") == "" {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsulAs", errors.New("CONSUL_HTTP_ADDR is not set")))
			return
		}

		path = os.ExpandEnv(path)

		l, err := source.NewConsul(path, decoder, nil)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsulAs", err))
			return
		}

		cfg.sources = append(cfg.sources, l)
	}
}

// WithConsulOptional returns an Option that adds a Consul source only when
// CONSUL_HTTP_ADDR is set.
// If CONSUL_HTTP_ADDR is not set, this option is a no-op (no source
// added, no error).
// Use this for development without Consul; use WithConsul when Consul is
// required and should fail at construction if env is missing.
//
// The format is automatically detected from the path extension. Paths
// support environment variable expansion (${VAR} or $VAR).
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithConsulOptional("production/service.yaml"),  // No-op when CONSUL_HTTP_ADDR is unset
//	)
func WithConsulOptional(path string) Option {
	return func(cfg *config) {
		if os.Getenv("CONSUL_HTTP_ADDR") == "" {
			return
		}

		path = os.ExpandEnv(path)

		c, err := detectFormat(path)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsulOptional", err))
			return
		}

		l, err := source.NewConsul(path, c, nil)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsulOptional", err))
			return
		}

		cfg.sources = append(cfg.sources, l)
	}
}

// WithConsulAsOptional returns an Option that adds a Consul source with
// explicit decoder only when CONSUL_HTTP_ADDR is set.
// If CONSUL_HTTP_ADDR is not set, this option is a no-op (no source
// added, no error).
// Use this for development without Consul; use WithConsulAs when Consul
// is required and should fail at construction if env is missing.
//
// Paths support environment variable expansion (${VAR} or $VAR).
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithConsulAsOptional("production/service", codec.JSON()),
//	)
func WithConsulAsOptional(path string, decoder codec.Decoder) Option {
	return func(cfg *config) {
		if os.Getenv("CONSUL_HTTP_ADDR") == "" {
			return
		}

		path = os.ExpandEnv(path)

		l, err := source.NewConsul(path, decoder, nil)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsulAsOptional", err))
			return
		}

		cfg.sources = append(cfg.sources, l)
	}
}

// WithContent returns an Option that configures the Synthra instance to
// load configuration data from a byte slice.
// The decoder parameter specifies how to decode the data (e.g.,
// codec.YAML(), codec.JSON()).
//
// Example:
//
//	yamlContent := []byte("server:\n  port: 8080")
//	cfg := synthra.MustNew(
//	    synthra.WithContent(yamlContent, codec.YAML()),
//	)
func WithContent(data []byte, decoder codec.Decoder) Option {
	return func(cfg *config) {
		cfg.sources = append(cfg.sources, source.NewFileContent(data, decoder))
	}
}

// WithFileDumperAs returns an Option that configures the Synthra instance
// to dump configuration data to a file with explicit encoder.
// Use this when the file doesn't have an extension or when you need to
// override the format detection.
//
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
// Example: "${OUTPUT_DIR}/config" expands to "/tmp/config" when OUTPUT_DIR=/tmp
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithFileDumperAs("output", codec.YAML()),  // No extension, specify YAML
//	)
func WithFileDumperAs(path string, encoder codec.Encoder) Option {
	return func(cfg *config) {
		path = os.ExpandEnv(path)
		cfg.dumpers = append(cfg.dumpers, dumper.NewFile(path, encoder))
	}
}

// WithBinding returns an Option that configures the Synthra instance to
// bind configuration data to a struct.
func WithBinding(v any) Option {
	return func(cfg *config) {
		if v == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithBinding", errors.New("binding target cannot be nil")))
			return
		}
		if reflect.TypeOf(v).Kind() != reflect.Pointer {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithBinding", errors.New("binding target must be a pointer")))
			return
		}
		cfg.binding = v
	}
}

// WithTag sets a custom struct tag name for binding (default: "synthra").
// Use it when the default tag clashes with another convention or you want
// a shorter key (for example "cfg" or "config").
//
// Example:
//
//	type AppConfig struct {
//	    Port int `cfg:"port"` // Using custom tag
//	}
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithBinding(&appConfig),
//	    synthra.WithTag("cfg"),
//	)
func WithTag(tagName string) Option {
	return func(cfg *config) {
		if tagName == "" {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithTag", errors.New("tag name cannot be empty")))
			return
		}
		cfg.tagName = tagName
	}
}

// WithJSONSchema adds a JSON Schema for validation.
func WithJSONSchema(schema []byte) Option {
	return func(cfg *config) {
		// Use a unique schema name to avoid caching issues
		//nolint:gosec // rand.Int() is used for a unique schema name, not security sensitive
		schemaName := fmt.Sprintf("inline_%d.json", rand.Int())
		compiler := jsonschema.NewCompiler()

		jsonSchema, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithJSONSchema", err))
			return
		}

		if err = compiler.AddResource(schemaName, jsonSchema); err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithJSONSchema", err))
			return
		}
		s, err := compiler.Compile(schemaName)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithJSONSchema", err))
			return
		}
		cfg.jsonSchemaCompiled = s
	}
}

// WithValidator adds a custom validation function.
func WithValidator(fn func(map[string]any) error) Option {
	return func(cfg *config) {
		if fn == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithValidator", errors.New("validator cannot be nil")))
			return
		}
		cfg.customValidators = append(cfg.customValidators, fn)
	}
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
		values:             &map[string]any{},
		sources:            cfg.sources,
		dumpers:            cfg.dumpers,
		binding:            cfg.binding,
		tagName:            cfg.tagName,
		jsonSchemaCompiled: cfg.jsonSchemaCompiled,
		customValidators:   cfg.customValidators,
	}
}

// New creates a new Synthra instance with the provided options.
// Options are applied to an internal config; after validation, the public
// Synthra is built from it.
// Options are applied in order; validation errors are collected and
// reported after all options are applied, so callers never receive a
// partially-initialized config. Options must not be nil—
// passing a nil option results in a validation error. Use MustNew for
// main() or when panic on error is acceptable.
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

// MustNew creates a new Synthra instance with the provided options.
// It panics if validation fails after applying options.
// Use this in main() or initialization code where panic is acceptable.
// For cases where error handling is needed, use New() instead.
func MustNew(opts ...Option) *Synthra {
	cfg, err := New(opts...)
	if err != nil {
		panic(fmt.Sprintf("synthra: validation failed: %v", err))
	}
	return cfg
}

// Validator is an interface for structs that can validate their own configuration.
// The validation package uses the same contract (validation.Validator); a
// type implementing either satisfies both.
type Validator interface {
	Validate() error
}

// applyDefaults applies default values from struct tags to a struct.
// It walks through the struct fields and sets defaults for fields that
// have the 'default' tag and are currently zero-valued.
func applyDefaults(target any) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer {
		return fmt.Errorf("target must be a pointer")
	}

	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	return setDefaults(val)
}

// setDefaults recursively sets default values on a struct.
func setDefaults(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := setDefaults(field); err != nil {
				return err
			}
			continue
		}

		// Check if field has a default tag
		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			continue
		}

		// Only set default if field is zero-valued
		if !isZeroValue(field) {
			continue
		}

		// Set the default value based on field type
		if err := setDefaultValue(field, defaultTag); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// isZeroValue checks if a [reflect.Value] is the zero value for its type.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return false
	}
}

// setDefaultValue sets a default value on a field based on its type.
func setDefaultValue(field reflect.Value, defaultVal string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special handling for time.Duration
		if field.Type() == reflect.TypeFor[time.Duration]() {
			d, err := time.ParseDuration(defaultVal)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
		} else {
			i, err := cast.ToInt64E(defaultVal)
			if err != nil {
				return err
			}
			field.SetInt(i)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := cast.ToUint64E(defaultVal)
		if err != nil {
			return err
		}
		field.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := cast.ToFloat64E(defaultVal)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := cast.ToBoolE(defaultVal)
		if err != nil {
			return err
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("unsupported type for default tag: %s", field.Kind())
	}
	return nil
}

// getDecoderConfig returns a cached decoder configuration to reduce
// reflection overhead.
func (c *Synthra) getDecoderConfig() *mapstructure.DecoderConfig {
	c.decoderOnce.Do(func() {
		tagName := c.tagName
		if tagName == "" {
			tagName = "synthra" // Fallback to default
		}
		c.decoderConfig = &mapstructure.DecoderConfig{
			TagName:          tagName,
			Squash:           true,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
				mapstructure.StringToTimeHookFunc(time.RFC3339),
				mapstructure.StringToURLHookFunc(),
			),
		}
	})
	return c.decoderConfig
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
// Errors:
//   - Returns [*ConfigError] with [OpLoad] if ctx is nil ([ErrNilContext])
//   - Returns [*ConfigError] with [OpLoad] if any source fails to load or merge
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

	if c.jsonSchemaCompiled != nil {
		if err = c.jsonSchemaCompiled.Validate(newValues); err != nil {
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

// decodeBindingInto decodes values into target using mapstructure. Errors
// match the messages produced by the former bind/bindAndValidate helpers.
func (c *Synthra) decodeBindingInto(target, values any) error {
	decoderCfg := c.getDecoderConfig()
	decoderCfg.Result = target

	decoder, err := mapstructure.NewDecoder(decoderCfg)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err = decoder.Decode(values); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
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

// String returns the value at key as a string.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	host, err := cfg.String("server.host")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) String(key string) (string, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return "", err
	}
	s, err := cast.ToStringE(v)
	if err != nil {
		return "", NewConfigError(OpGet, key, err)
	}
	return s, nil
}

// Int returns the value at key as an int.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	port, err := cfg.Int("server.port")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Int(key string) (int, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	i, err := cast.ToIntE(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return i, nil
}

// Int64 returns the value at key as an int64.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	maxSize, err := cfg.Int64("max_size")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Int64(key string) (int64, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	i, err := cast.ToInt64E(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return i, nil
}

// Float64 returns the value at key as a float64.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	rate, err := cfg.Float64("rate")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Float64(key string) (float64, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	f, err := cast.ToFloat64E(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return f, nil
}

// Bool returns the value at key as a bool.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	debug, err := cfg.Bool("debug")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Bool(key string) (bool, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return false, err
	}
	b, err := cast.ToBoolE(v)
	if err != nil {
		return false, NewConfigError(OpGet, key, err)
	}
	return b, nil
}

// Duration returns the value at key as a [time.Duration].
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	timeout, err := cfg.Duration("timeout")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Duration(key string) (time.Duration, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return 0, err
	}
	d, err := cast.ToDurationE(v)
	if err != nil {
		return 0, NewConfigError(OpGet, key, err)
	}
	return d, nil
}

// Time returns the value at key as a [time.Time].
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	startTime, err := cfg.Time("start_time")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) Time(key string) (time.Time, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return time.Time{}, err
	}
	tm, err := cast.ToTimeE(v)
	if err != nil {
		return time.Time{}, NewConfigError(OpGet, key, err)
	}
	return tm, nil
}

// StringSlice returns the value at key as a []string.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	tags, err := cfg.StringSlice("tags")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) StringSlice(key string) ([]string, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return nil, err
	}
	s, err := cast.ToStringSliceE(v)
	if err != nil {
		return nil, NewConfigError(OpGet, key, err)
	}
	return s, nil
}

// IntSlice returns the value at key as a []int.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	ports, err := cfg.IntSlice("ports")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) IntSlice(key string) ([]int, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return nil, err
	}
	s, err := cast.ToIntSliceE(v)
	if err != nil {
		return nil, NewConfigError(OpGet, key, err)
	}
	return s, nil
}

// StringMap returns the value at key as a map[string]any.
// It returns an error if c is nil, the key is missing,
// or the value cannot be converted.
//
// Example:
//
//	metadata, err := cfg.StringMap("metadata")
//	if err != nil {
//	    return err
//	}
func (c *Synthra) StringMap(key string) (map[string]any, error) {
	v, err := c.requireValue(key)
	if err != nil {
		return nil, err
	}
	m, err := cast.ToStringMapE(v)
	if err != nil {
		return nil, NewConfigError(OpGet, key, err)
	}
	return m, nil
}

// StringOr returns the value associated with the given key as a string,
// or the default value if not found.
//
// Example:
//
//	host := cfg.StringOr("server.host", "localhost")
func (c *Synthra) StringOr(key, defaultVal string) string {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToString(val)
}

// IntOr returns the value associated with the given key as an int, or
// the default value if not found.
//
// Example:
//
//	port := cfg.IntOr("server.port", 8080)
func (c *Synthra) IntOr(key string, defaultVal int) int {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToInt(val)
}

// Int64Or returns the value associated with the given key as an int64,
// or the default value if not found.
//
// Example:
//
//	maxSize := cfg.Int64Or("max_size", 1024)
func (c *Synthra) Int64Or(key string, defaultVal int64) int64 {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToInt64(val)
}

// Float64Or returns the value associated with the given key as a float64,
// or the default value if not found.
//
// Example:
//
//	rate := cfg.Float64Or("rate", 0.5)
func (c *Synthra) Float64Or(key string, defaultVal float64) float64 {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToFloat64(val)
}

// BoolOr returns the value associated with the given key as a boolean,
// or the default value if not found.
//
// Example:
//
//	debug := cfg.BoolOr("debug", false)
func (c *Synthra) BoolOr(key string, defaultVal bool) bool {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToBool(val)
}

// DurationOr returns the value associated with the given key as a
// [time.Duration], or the default value if not found.
//
// Example:
//
//	timeout := cfg.DurationOr("timeout", 30*time.Second)
func (c *Synthra) DurationOr(key string, defaultVal time.Duration) time.Duration {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToDuration(val)
}

// TimeOr returns the value associated with the given key as a [time.Time],
// or the default value if not found.
//
// Example:
//
//	startTime := cfg.TimeOr("start_time", time.Now())
func (c *Synthra) TimeOr(key string, defaultVal time.Time) time.Time {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToTime(val)
}

// StringSliceOr returns the value associated with the given key as a
// slice of strings, or the default value if not found.
//
// Example:
//
//	tags := cfg.StringSliceOr("tags", []string{"default"})
func (c *Synthra) StringSliceOr(key string, defaultVal []string) []string {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToStringSlice(val)
}

// IntSliceOr returns the value associated with the given key as a slice
// of integers, or the default value if not found.
//
// Example:
//
//	ports := cfg.IntSliceOr("ports", []int{8080, 8081})
func (c *Synthra) IntSliceOr(key string, defaultVal []int) []int {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToIntSlice(val)
}

// StringMapOr returns the value associated with the given key as a
// map[string]any, or the default value if not found.
//
// Example:
//
//	metadata := cfg.StringMapOr("metadata", map[string]any{"version": "1.0"})
func (c *Synthra) StringMapOr(key string, defaultVal map[string]any) map[string]any {
	if c == nil {
		return defaultVal
	}
	val := c.Get(key)
	if val == nil {
		return defaultVal
	}
	return cast.ToStringMap(val)
}
