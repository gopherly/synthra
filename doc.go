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

// Package synthra synthesizes configuration for Go applications from many
// sources into one coherent runtime state.
//
// The name follows σύνθεσις (synthesis): to put together, to compose into a whole.
// Modern systems are configured in layers—files, environment variables, defaults,
// flags, secret stores, and remote providers. Each layer is incomplete alone;
// Synthra merges them in order (later overrides earlier), validates, binds to
// structs, and exposes the result through [Synthra].
// **From many sources, one state.**
//
// Keys are case-insensitive; access uses dot notation.
//
// The package uses the same functional options pattern as other Gopherly packages:
// options apply to an internal config struct, and the constructor validates and
// builds the public [Synthra] from it. The returned [Synthra] is the runtime
// object used for Load, Get, and Dump.
//
// # Key Features
//
//   - Multiple configuration sources (files, [io/fs.FS], environment variables,
//     Consul)
//   - Automatic format detection and decoding (JSON, YAML, TOML)
//   - JSON Schema defaults: "default" values declared in the schema are
//     automatically applied to missing keys, including patternProperties
//   - Dynamic schema selection ([WithJSONSchemaSelector]) for version-based or
//     content-based schema routing at Load time
//   - Post-load transforms ([WithTransform]) and POSIX-style variable
//     substitution ([WithEnvSubst]) run before validation
//   - Composable variable resolvers ([gopherly.dev/synthra/resolve]) for
//     maps, OS environment variables, and prefixed env vars
//   - Struct binding with automatic type conversion
//   - Validation using JSON Schema or custom validators
//   - Case-insensitive key access with dot notation
//   - Thread-safe configuration loading and access
//   - Configuration dumping to files or custom destinations
//
// # Quick Start
//
// Create a configuration instance with sources. Options are applied in order;
// any validation errors are reported when the config is built (by New or MustNew).
// Options must not be nil; passing a nil option results in a validation error.
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnv("APP_"),
//	)
//
// Load the configuration:
//
//	if err := cfg.Load(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// Access configuration values (strict reads return an error if the key is
// missing or the value cannot be coerced; use *Or methods for defaults):
//
//	port, err := cfg.Int("server.port")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	host := cfg.StringOr("server.host", "localhost")
//	debug, err := cfg.Bool("debug")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Configuration Sources
//
// The package supports multiple configuration sources that can be combined:
//
// Files with automatic format detection:
//
//	synthra.WithFile("config.yaml")     // Detects YAML
//	synthra.WithFile("config.json")     // Detects JSON
//	synthra.WithFile("config.toml")     // Detects TOML
//
// Files with explicit format:
//
//	synthra.WithFileAs("config", codec.YAML)
//
// Virtual files inside an [io/fs.FS] (tests, [embed.FS], etc.):
//
//	synthra.WithFileFS(fsys, "config.yaml")
//	synthra.WithFileFSAs(fsys, "config", codec.YAML)
//
// Environment variables with prefix:
//
//	synthra.WithEnv("APP_")  // Loads APP_SERVER_PORT as server.port
//
// Consul key-value store (CONSUL_HTTP_ADDR required; construction fails if unset):
//
//	synthra.WithConsul("production/service.yaml")
//
// Conditional Consul (e.g. for local dev without Consul):
//
//	synthra.WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "",
//	    synthra.WithConsul("production/service.yaml"),
//	)
//
// Raw content:
//
//	yamlData := []byte("port: 8080")
//	synthra.WithContent(yamlData, codec.YAML)
//
// # Struct Binding
//
// Bind configuration to a struct for type-safe access:
//
//	type AppConfig struct {
//	    Port    int           `synthra:"port"`
//	    Host    string        `synthra:"host"`
//	    Timeout time.Duration `synthra:"timeout"`
//	    Debug   bool          `synthra:"debug" default:"false"`
//	}
//
//	var appConfig AppConfig
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithBinding(&appConfig),
//	)
//
//	if err := cfg.Load(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access typed fields directly
//	fmt.Printf("Server: %s:%d\n", appConfig.Host, appConfig.Port)
//
// # Validation
//
// Validate configuration using struct methods:
//
//	type Config struct {
//	    Port int `synthra:"port"`
//	}
//
//	func (c *Config) Validate() error {
//	    if c.Port < 1 || c.Port > 65535 {
//	        return fmt.Errorf("port must be between 1 and 65535")
//	    }
//	    return nil
//	}
//
// Validate using JSON Schema (also applies "default" values automatically):
//
//	schema := []byte(`{
//	    "type": "object",
//	    "properties": {
//	        "port": {"type": "integer", "minimum": 1, "maximum": 65535, "default": 8080}
//	    },
//	    "required": ["port"]
//	}`)
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithJSONSchema(schema), // validates AND fills in "default" values
//	)
//	// If config.yaml omits "port", Load sets it to 8080 before validating.
//
// Use [WithJSONSchemaSelector] when the schema to use depends on a value inside
// the config itself — for example an apiVersion field:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("manifest.yaml"),
//	    synthra.WithJSONSchemaSelector(func(values map[string]any) ([]byte, error) {
//	        version, ok := values["apiversion"].(string)
//	        if !ok || version == "" {
//	            return nil, errors.New("apiVersion is required")
//	        }
//	        return schemaRegistry.Get(version)
//	    }),
//	)
//	// The selector is called at Load time with the merged values, so it can
//	// branch on any config value. The selected schema applies defaults and
//	// validates exactly like WithJSONSchema. WithJSONSchema and
//	// WithJSONSchemaSelector are mutually exclusive.
//
// # Transforms and Variable Substitution
//
// [WithTransform] registers a function that processes the merged values after
// schema defaults and before validation. Multiple transforms run as a pipeline.
//
// [WithEnvSubst] is a convenience transform that expands POSIX-style ${VAR}
// placeholders in all string values. It supports the full POSIX substitution
// syntax: ${VAR:-default}, ${VAR:=default}, ${VAR^^}, ${VAR#pattern}, and more.
//
// [WithEnv] and [WithEnvSubst] solve different problems and work well together:
//
//   - [WithEnv] is a source. It reads environment variables and adds them to
//     the config map. For example, APP_SERVER_PORT=8080 becomes server.port.
//   - [WithEnvSubst] is a transform. It expands ${VAR} placeholders that are
//     already present in string values loaded from files or other sources.
//
// Example — expand ${ENV} from a static map:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithJSONSchema(schema),
//	    synthra.WithEnvSubst(resolve.Vars(map[string]string{
//	        "ENV":    "production",
//	        "REGION": "eu-west-1",
//	    })),
//	)
//	// If config.yaml has: envFile: ".env.${ENV}"
//	// After Load: cfg.Get("envfile") => ".env.production"
//
// Example — layer multiple resolvers (last wins):
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("deployah.yaml"),
//	    synthra.WithEnvSubst(
//	        resolve.Vars(manifestVars),    // lowest priority
//	        resolve.Vars(envFileVars),     // medium priority
//	        resolve.OSPrefix("DPY_VAR_"),  // highest priority
//	    ),
//	)
//	// config.yaml: port: ${PORT:-3000}
//	// If DPY_VAR_PORT=9090 is set in the environment, port becomes "9090".
//
// Example — custom transform to normalize values:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithTransform(func(values map[string]any) (map[string]any, error) {
//	        if level, ok := values["log_level"].(string); ok {
//	            values["log_level"] = strings.ToLower(level)
//	        }
//	        return values, nil
//	    }),
//	    synthra.WithJSONSchema(schema),
//	)
//
// Validate using custom functions:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithValidator(func(values map[string]any) error {
//	        if port, ok := values["port"].(int); ok && port < 1 {
//	            return fmt.Errorf("invalid port: %d", port)
//	        }
//	        return nil
//	    }),
//	)
//
// # Accessing Configuration Values
//
// Type-specific methods return (value, error). Missing keys and failed
// coercions are errors; use [errors.Is] with [ErrKeyNotFound] or [ErrNilConfig]
// as needed. Methods on a nil [*Synthra] return [ErrNilConfig].
//
//	// Basic types (strict)
//	port, err := cfg.Int("server.port")
//	if err != nil {
//	    return err
//	}
//	host, err := cfg.String("server.host")
//	if err != nil {
//	    return err
//	}
//	debug, err := cfg.Bool("debug")
//	if err != nil {
//	    return err
//	}
//	rate, err := cfg.Float64("rate")
//	if err != nil {
//	    return err
//	}
//
//	// Optional keys with defaults (no error when missing)
//	host := cfg.StringOr("server.host", "localhost")
//	port := cfg.IntOr("server.port", 8080)
//
//	// Collections (strict)
//	tags, err := cfg.StringSlice("tags")
//	if err != nil {
//	    return err
//	}
//	ports, err := cfg.IntSlice("ports")
//	if err != nil {
//	    return err
//	}
//	metadata, err := cfg.StringMap("metadata")
//	if err != nil {
//	    return err
//	}
//
//	// Time-related (strict)
//	timeout, err := cfg.Duration("timeout")
//	if err != nil {
//	    return err
//	}
//	startTime, err := cfg.Time("start_time")
//	if err != nil {
//	    return err
//	}
//
// Generic [Get] for typed reads (same missing-key errors; primitive coercion
// matches [GetOr] for unsupported kinds):
//
//	port, err := synthra.Get[int](cfg, "server.port")
//	if err != nil {
//	    log.Fatalf("port configuration required: %v", err)
//	}
//
// # Configuration Dumping
//
// Save the current configuration to a file:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithFileDumper("output.yaml"),
//	)
//
//	cfg.Load(context.Background())
//	cfg.Dump(context.Background())  // Writes to output.yaml
//
// # Thread Safety
//
// Synthra is safe for concurrent use by multiple goroutines.
// Configuration loading and reading are protected by internal locks.
// Multiple goroutines can safely call Load() and access configuration
// values simultaneously.
//
// # Escape hatches
//
// For debugging or custom serialization, [*Synthra.Values] returns a shallow
// copy of the merged top-level map. Nested maps, slices, and pointers are not
// deep-copied; do not mutate nested values—treat the snapshot as read-only.
//
// # Error Handling
//
// Construction failures, load/dump failures, and accessor type-conversion
// failures are returned as [*ConfigError], shaped like [os.PathError]:
// Op names the entrypoint ([OpNew], [OpLoad], [OpDump], or [OpGet]); Path is
// a diagnostic locator whose meaning depends on Op; Err is the cause for
// [errors.Unwrap], [errors.Is], and [errors.As].
//
// Use [errors.As] to inspect the structured error and switch on Op:
//
//	if err := cfg.Load(ctx); err != nil {
//	    var ce *synthra.ConfigError
//	    if errors.As(err, &ce) {
//	        switch ce.Op {
//	        case synthra.OpLoad:
//	            log.Error("load failed", "path", ce.Path, "err", ce.Err)
//	        }
//	    }
//	    return err
//	}
//
// Use [errors.Is] for fixed outcomes such as a missing key, nil receiver,
// or nil context:
//
//	_, err := cfg.Int("server.port")
//	if errors.Is(err, synthra.ErrKeyNotFound) {
//	    return useDefaultPort()
//	}
//
// [New] may return [errors.Join] of multiple [*ConfigError] values. A single
// [errors.As] finds the first in the tree; to log every construction error,
// iterate using the [errors.Join] unwrap slice (see [errors.Join]).
//
// # Examples
//
// See the examples directory for complete working examples demonstrating
// various configuration patterns and use cases including:
//
//   - examples/basic — file loading and struct binding
//   - examples/environment — environment-only configuration
//   - examples/webapp — layered YAML + env, binding, and Validate
//   - examples/jsonschema — JSON Schema validation
//   - examples/jsonschema-defaults — JSON Schema defaults and WithEnvSubst
//   - examples/customvalidator — custom validation functions
//   - examples/dump — configuration dumping
//   - examples/consul — optional Consul integration
//
// For more details, see the package documentation at
// https://pkg.go.dev/gopherly.dev/synthra
package synthra
