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
// Modern systems are configured in layers: files, environment variables, defaults,
// flags, secret stores, and remote providers. Each layer is incomplete alone;
// Synthra merges them in order (later overrides earlier), validates, binds to
// structs, and exposes the result through [Synthra].
// **From many sources, one state.**
//
// # Key casing
//
// Synthra keeps the casing your config sources use. If your file says
// `apiVersion`, the loaded map will have `apiVersion` too. This applies to all
// sources (YAML, JSON, TOML, Consul, embedded files, inline content, and any
// custom source you write). Only the matching is case-insensitive: you can read
// the same key as `apiVersion` or `APIVERSION` and get the same value.
//
// The one exception is environment variables. Environment variables are
// uppercase by convention (`APP_API_VERSION`), so the env source lowercases
// them to produce a nested map. A [WithEnv]("APP_") source always contributes
// lowercase keys like `apiversion`. When env meets another source that already
// has the same key in a different casing, the case-insensitive merge keeps the
// first source's casing and overrides only the value. So if your YAML says
// `apiVersion: v1` and `APP_APIVERSION=v2` is set, the final map has
// `apiVersion: v2`.
//
// When two non-env sources use different casings for the same key, the first
// source wins for the name and the last source wins for the value. So if
// `base.yaml` has `ApiVersion: v1` and `override.yaml` has `apiVersion: v2`,
// the final map looks like `ApiVersion: v2`. The typo in the base file is
// preserved.
//
// To avoid that, register a JSON Schema. Before validation runs, Synthra renames
// any case-different keys in the data to match the schema. So `ApiVersion: v2`
// becomes `apiVersion: v2` if your schema says
// `"properties": {"apiVersion": ...}`.
//
//	cfg.Get("apiVersion") == cfg.Get("apiversion")  // both work
//
//	// base.yaml -> ApiVersion: v1
//	// override.yaml -> apiVersion: v2
//	// result: ApiVersion: v2  (first writer's casing wins)
//
//	// Same files, with a schema declaring apiVersion:
//	// result: apiVersion: v2  (schema is the authority)
//
//	// config.yaml -> apiVersion: v1
//	// APP_APIVERSION=v2
//	// result: apiVersion: v2  (YAML casing wins, env overrides value)
//
// Keys without a schema declaration keep whatever casing the first source
// provided. The env source always produces lowercase keys. `patternProperties`
// and `additionalProperties` dynamic keys are not renamed by the schema. Keys
// inside list elements are only renamed when the schema declares an `items`
// object for that list.
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
//   - Pipeline processing: schema steps, transforms, and validators are
//     executed in registration order, enabling multi-phase workflows
//   - JSON Schema defaults: "default" values declared in the schema are
//     automatically applied to missing keys, including patternProperties
//   - Dynamic schema selection ([WithJSONSchemaFunc]) for version-based or
//     content-based schema routing at Load time
//   - POSIX-style variable substitution ([WithEnvSubst]) and arbitrary
//     transforms ([WithTransform]) can be interleaved with schema steps
//   - Composable variable resolvers ([FromMap], [FromEnv], [FromEnvFile]) for
//     maps, OS env, and .env files; prefix stripping via [Resolver.Prefix] and
//     first-wins fallback chains via [Resolver.Or]
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
// # Pipeline
//
// After all sources are merged, Synthra executes pipeline steps in the order
// they were registered. Steps are added by:
//
//   - [WithJSONSchema]: validates against a static schema and applies its
//     declared default values.
//   - [WithJSONSchemaFunc]: same as [WithJSONSchema], but the schema bytes are
//     returned by a callback that receives the current values map. Use this when
//     the schema depends on a value inside the config (e.g. an apiVersion field).
//   - [WithTransform]: arbitrary map mutation step.
//   - [WithEnvSubst]: convenience transform that expands ${VAR} placeholders
//     using a [Resolver]. Compose multiple sources with [Resolver.Or] (first
//     match wins; see "Resolver vs Source precedence" below).
//   - [WithEnvSubstFunc]: same as [WithEnvSubst], but the Resolver is built by
//     a callback at Load time. Use this when the resolver depends on a value
//     already loaded from a source (e.g. a .env file path in the config file).
//   - [WithValidator]: read-only check that may return an error.
//
// Because steps run in registration order, you can interleave them freely.
// A common pattern is two-phase validation: validate partial data before
// substitution, substitute, then validate the final form.
//
// Example: dynamic schema selection based on apiVersion:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("manifest.yaml"),
//	    synthra.WithJSONSchemaFunc(func(v *synthra.Values) ([]byte, error) {
//	        version, err := v.String("apiVersion")
//	        if err != nil || version == "" {
//	            return nil, errors.New("apiVersion is required")
//	        }
//	        return schemaRegistry.Get(version)
//	    }),
//	)
//
// Example: two-phase validation (validate before and after substitution):
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("manifest.yaml"),
//	    // Step 1: validate the raw "environments" block before substitution.
//	    synthra.WithJSONSchemaFunc(environmentsSchema),
//	    // Step 2: expand ${VAR} placeholders.
//	    synthra.WithEnvSubst(synthra.FromEnv()),
//	    // Step 3: validate the fully-substituted manifest.
//	    synthra.WithJSONSchemaFunc(manifestSchema),
//	)
//
// Multiple [WithJSONSchema] and [WithJSONSchemaFunc] calls are fully supported
// and each adds an independent schema step at the point it was registered. There
// is no mutual-exclusivity restriction.
//
// # Resolver vs Source precedence
//
// Synthra uses two different precedence rules, one for each kind of operation:
//
//   - Sources (tree merge): later wins. When you call [WithFile], [WithEnv],
//     or [WithSource] multiple times, each call layers on top of the previous
//     one. The last source to provide a key wins. This matches every major
//     config library (viper, koanf, dynaconf, Figment .merge).
//
//   - Resolvers (per-key lookup): first wins. [Resolver.Or] tries the receiver
//     before each fallback, returning as soon as one reports found=true. This
//     matches stdlib [context.Value], where the innermost (highest-priority)
//     context shadows outer ones, and other per-key lookup chains (Spring
//     PropertySource, [os/exec.LookPath]).
//
// These rules are different because the operations are different: overlaying a
// full tree is not the same as looking up one key in a chain of stores.
// Each rule is named for its operation so you do not need to remember which
// library uses which: Sources layer in registration order (last wins),
// Resolvers fall through in call order (first wins via Or).
//
// [WithEnv] and [WithEnvSubst] solve different problems and work well together:
//
//   - [WithEnv] is a source. It reads environment variables and adds them to
//     the config map. For example, APP_SERVER_PORT=8080 becomes server.port.
//   - [WithEnvSubst] is a transform. It expands ${VAR} placeholders that are
//     already present in string values loaded from files or other sources.
//
// Example: three-layer priority with Or (highest priority first):
//
//	envFile, err := synthra.FromEnvFile(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("deployah.yaml"),
//	    synthra.WithEnvSubst(
//	        synthra.FromEnv().Prefix("DPY_VAR_").  // highest: prefixed OS env
//	            Or(envFile).                         // middle:  .env file
//	            Or(synthra.FromMap(manifestVars)),   // lowest:  static defaults
//	    ),
//	)
//	// config.yaml: port: ${PORT:-3000}
//	// If DPY_VAR_PORT=9090 is set in the environment, port becomes "9090".
//
// Example: custom transform to normalize values before schema validation:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithTransform(func(v *synthra.Values) error {
//	        if level := v.StringOr("logLevel", ""); level != "" {
//	            return v.Set("logLevel", strings.ToLower(level))
//	        }
//	        return nil
//	    }),
//	    synthra.WithJSONSchema(schema), // validates the normalized values
//	)
//
// Validate using custom functions:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithValidator(func(v *synthra.Values) error {
//	        if port := v.IntOr("port", 0); port < 1 {
//	            return fmt.Errorf("invalid port: %d", port)
//	        }
//	        return nil
//	    }),
//	)
//
// Error paths for pipeline failures follow the pattern "step[N]:kind" where N
// is the zero-based step index and kind is "schema", "transform", or "validator".
//
// # Pipeline callbacks and Values
//
// Synthra's pipeline callbacks receive a [*Values] wrapper instead of a raw
// map. The wrapper gives you safe, case-insensitive, typed access to the data
// flowing through the pipeline.
//
// Direct map access is case-sensitive at the Go language level. If your YAML
// says `apiVersion` but the merged map ends up storing it under a slightly
// different casing, `values["apiVersion"]` returns nil. The [*Values] wrapper
// fixes that. All its methods are case-insensitive.
//
// Reading:
//
//	v.Get("metadata.name")         // any, case-insensitive
//	v.Has("server.tls.enabled")    // bool
//	v.String("apiVersion")         // (string, error)
//	v.IntOr("server.port", 8080)   // int with default
//
// Writing:
//
//	v.Set("metadata.region", "eu-west-1")  // creates intermediate maps
//	v.Delete("debug.experimental")
//
// Walking the tree:
//
//	v.Walk(func(path string, val any) (any, bool) {
//	    if s, ok := val.(string); ok && strings.HasPrefix(s, "${") {
//	        return strings.TrimPrefix(s, "${"), true
//	    }
//	    return val, false
//	})
//
// When you must hand the underlying map to code that expects a plain
// `map[string]any`, call `v.Raw()`. Mutations on the returned map are visible
// through the same [*Values].
//
// [WithTransform], [WithValidator], [WithEnvSubstFunc], and
// [WithJSONSchemaFunc] run at the map stage, before binding, on the merged
// [*Values].
//
// [OnBound] is a binding-scoped option that goes inside [WithBinding]. It runs
// at the binding stage: after the bound struct is decoded and defaults applied,
// but before its `Validate()` method (if it implements [Validator]). The type
// parameter is inferred from the closure, and the compiler enforces that it
// matches the binding target.
//
// Example combining both stages:
//
//	synthra.WithFile("config.yaml"),
//	synthra.WithTransform(func(v *synthra.Values) error {
//	    if v.StringOr("env", "dev") == "prod" {
//	        return v.Set("logging.level", "warn")
//	    }
//	    return nil
//	}),
//	synthra.WithBinding(&app,
//	    synthra.OnBound(func(a *App) error {
//	        a.Logging.Level = strings.ToLower(a.Logging.Level)
//	        return nil
//	    }),
//	),
//
// Because [OnBound] is a sub-option of [WithBinding], Go infers the same T for
// both. If the closure type does not match the binding target, you get a compile
// error, not a runtime panic:
//
//	var server Server
//	synthra.WithBinding(&server,
//	    synthra.OnBound(func(a *App) error { ... }),  // compile error
//	)
//
// [*Values] is not safe for concurrent use. Each Load creates its own; do not
// share one across goroutines. Only one [WithBinding] per Synthra instance is
// supported.
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
// deep-copied; do not mutate nested values. Treat the snapshot as read-only.
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
//   - examples/basic: YAML file and struct binding
//   - examples/webapp: YAML defaults, env overrides, binding, and Validate
//   - examples/testing: synthratest helpers and source.NewMap
//   - examples/schema: WithJSONSchema defaults and patternProperties
//   - examples/casing: case-insensitive merge and schema as casing authority
//   - examples/hooks: WithTransform, WithValidator, and OnBound[T] in one pipeline
//   - examples/codecs: WithFileAs (JSON, TOML) and WithFileDumperAs (YAML dump)
//   - examples/envsubst-layered: three-layer Resolver.Or precedence
//   - examples/multi-schema: two-phase validation with WithJSONSchemaFunc
//   - examples/consul: optional Consul source with WithIf
//
// For more details, see the package documentation at
// https://pkg.go.dev/gopherly.dev/synthra
package synthra
