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
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/fluxcd/pkg/envsubst"
	"gopherly.dev/synthra/codec"
	"gopherly.dev/synthra/dumper"
	"gopherly.dev/synthra/source"
)

// WithSource adds a custom [Source] to the configuration loader.
// Use it to plug in sources not covered by the built-in options
// (e.g. a database, remote API, or custom file format).
// The source must not be nil.
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithSource(myCustomSource),
//	)
func WithSource(loader Source) Option {
	return func(cfg *config) {
		if loader == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithSource", errors.New("source cannot be nil")))
			return
		}
		cfg.sources = append(cfg.sources, loader)
	}
}

// WithIf returns an Option that applies the provided options only when
// condition is true.
// When condition is false, this option is a no-op.
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "",
//	        synthra.WithConsul("production/service.yaml"),
//	    ),
//	)
func WithIf(condition bool, opts ...Option) Option {
	return func(cfg *config) {
		if !condition {
			return
		}
		for _, opt := range opts {
			opt(cfg)
		}
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

// WithDumper adds a custom [Dumper] to the configuration dumper.
// Use it to plug in dumpers not covered by the built-in options
// (e.g. a database, remote API, or custom file format).
// The dumper must not be nil.
//
// Example:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithDumper(myCustomDumper),
//	)
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

// WithFileFSAs returns an Option that loads configuration from path inside
// fsys using an explicit decoder. It combines [WithFileFS] (embedded
// filesystem) with [WithFileAs] (explicit decoder) for files that have no
// extension or need a format override.
//
// Paths support environment variable expansion using ${VAR} or $VAR syntax.
// If fsys is nil, [New] returns a validation error.
//
// Example:
//
//	//go:embed configs
//	var configFS embed.FS
//
//	cfg := synthra.MustNew(
//	    synthra.WithFileFSAs(configFS, "configs/app", codec.YAML),
//	)
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
// For conditional Consul (e.g., development without Consul), wrap this
// option with WithIf.
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
		c, err := detectFormat(path)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithConsul", err))
			return
		}
		addConsulSource(cfg, "WithConsul", path, c)
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
//	    synthra.WithFileAs("config", codec.YAML),      // No extension, specify YAML
//	    synthra.WithFileAs("config.dat", codec.JSON),  // Wrong extension, specify JSON
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
// For conditional Consul (e.g., development without Consul), wrap this
// option with WithIf.
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
//	    synthra.WithConsulAs("production/service", codec.JSON),
//	)
func WithConsulAs(path string, decoder codec.Decoder) Option {
	return func(cfg *config) {
		addConsulSource(cfg, "WithConsulAs", path, decoder)
	}
}

// consulNewSource is the constructor used by addConsulSource. It is a
// package-level variable so tests can replace it without a real Consul server.
var consulNewSource = func(path string, decoder codec.Decoder, kv source.ConsulKV) (Source, error) {
	return source.NewConsul(path, decoder, kv)
}

func addConsulSource(cfg *config, opName, path string, decoder codec.Decoder) {
	if os.Getenv("CONSUL_HTTP_ADDR") == "" {
		cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, opName, errors.New("CONSUL_HTTP_ADDR is not set")))
		return
	}

	path = os.ExpandEnv(path)

	l, err := consulNewSource(path, decoder, nil)
	if err != nil {
		cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, opName, err))
		return
	}

	cfg.sources = append(cfg.sources, l)
}

// WithContent returns an Option that configures the Synthra instance to
// load configuration data from a byte slice.
// The decoder parameter specifies how to decode the data (e.g.,
// codec.YAML, codec.JSON).
//
// Example:
//
//	yamlContent := []byte("server:\n  port: 8080")
//	cfg := synthra.MustNew(
//	    synthra.WithContent(yamlContent, codec.YAML),
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
//	    synthra.WithFileDumperAs("output", codec.YAML),  // No extension, specify YAML
//	)
func WithFileDumperAs(path string, encoder codec.Encoder) Option {
	return func(cfg *config) {
		path = os.ExpandEnv(path)
		cfg.dumpers = append(cfg.dumpers, dumper.NewFile(path, encoder))
	}
}

// BindingOption is the constraint type for options that may appear inside
// [WithBinding]. The T parameter ties each option to the bound target type
// so the compiler enforces alignment between the target and any hook.
//
// T appears in the [bindingConfig] parameter of applyToBinding so that two
// BindingOption instantiations with different T are distinct interfaces at
// compile time, not just at the type-checker. Without that, a method set
// independent of T would let BindingOption[App] satisfy BindingOption[Server]
// silently and the type-mismatch check would only happen at Load time.
type BindingOption[T any] interface {
	applyToBinding(b *bindingConfig[T])
}

// bindingConfig collects what [BindingOption] values apply at construction.
// It is package-internal; users see BindingOption[T] only through its public
// constructors. The generic parameter binds the configured hooks to the
// target type chosen by [WithBinding].
type bindingConfig[T any] struct {
	hooks []func(*T) error
}

// WithBinding registers target as the destination for struct decoding and
// accepts binding-scoped options. T is inferred from target; every option
// passed must share the same T, which the compiler enforces.
//
// Example:
//
//	type Config struct {
//	    Server struct {
//	        Host string `synthra:"host"`
//	        Port int    `synthra:"port"`
//	    } `synthra:"server"`
//	}
//
//	var appCfg Config
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithBinding(&appCfg,
//	        synthra.OnBound(func(c *Config) error {
//	            c.Server.Host = strings.ToLower(c.Server.Host)
//	            return nil
//	        }),
//	    ),
//	)
//	fmt.Println(appCfg.Server.Port) // populated from config
func WithBinding[T any](target *T, opts ...BindingOption[T]) Option {
	return func(cfg *config) {
		if target == nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithBinding", errors.New("binding target cannot be nil")))
			return
		}
		bc := &bindingConfig[T]{}
		for _, opt := range opts {
			opt.applyToBinding(bc)
		}
		cfg.binding = target
		// Adapt the typed hooks into the untyped storage shape used by Load.
		// The defensive *T assertion can only fail if a caller stores a hook
		// from a different type's bindingConfig — impossible through the
		// public API since BindingOption[T] now genuinely depends on T.
		cfg.bindingHooks = make([]func(any) error, 0, len(bc.hooks))
		for _, h := range bc.hooks {
			cfg.bindingHooks = append(cfg.bindingHooks, func(target any) error {
				t, ok := target.(*T)
				if !ok {
					return fmt.Errorf("OnBound: unexpected target type %T", target)
				}
				return h(t)
			})
		}
	}
}

// OnBound registers a function that runs against the bound struct after
// decode and applyDefaults, but before [Validator.Validate].
//
// Use OnBound for type-safe normalization — lowercasing log levels,
// computing derived fields, applying region-based defaults. For map-level
// work before binding, use [WithTransform].
//
// The function must not be nil. Multiple OnBound hooks run in registration
// order; the first error stops the pipeline.
func OnBound[T any](fn func(*T) error) BindingOption[T] {
	return onBoundOption[T]{fn: fn}
}

type onBoundOption[T any] struct {
	fn func(*T) error
}

// Compile-time check: onBoundOption satisfies BindingOption.
var _ BindingOption[struct{}] = onBoundOption[struct{}]{}

func (o onBoundOption[T]) applyToBinding(b *bindingConfig[T]) { //nolint:unused
	if o.fn == nil {
		// Defer the nil-check to Load so it surfaces as a typed *ConfigError
		// with the binding-hook[N] path, matching how nil functions are
		// reported elsewhere in the pipeline.
		b.hooks = append(b.hooks, func(_ *T) error {
			return errors.New("OnBound: function cannot be nil")
		})
		return
	}
	b.hooks = append(b.hooks, o.fn)
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

// WithJSONSchema adds a static JSON Schema as a pipeline step. The schema is
// used to apply default values and validate the configuration at the point in
// the pipeline where this option was registered.
//
// Synthra supports JSON Schema drafts 4, 6, 7, 2019-09, and 2020-12.
//
// # Automatic defaults
//
// Synthra extracts every "default" value declared in the schema and applies it
// to any key that is missing from the current values map. Defaults are applied
// before validation, so the schema validator always sees a fully populated map.
//
// Defaults are applied recursively at every level:
//   - "properties" — fills missing fixed-name keys in an object
//   - "patternProperties" — fills missing keys inside every existing map entry
//     whose name matches the regular-expression pattern
//   - "items" — fills missing keys inside each element of an array
//
// User-provided values are never overridden; only absent keys are filled.
//
// # Validation
//
// After defaults are applied the values are validated against the schema.
// If validation fails, Load returns a [*ConfigError] with Op [OpLoad] and
// Path "step[N]:schema" where N is the zero-based index of this step in the
// registered pipeline.
//
// Multiple [WithJSONSchema] calls are allowed and each adds an independent
// pipeline step that runs at the point it was registered.
//
// Example:
//
//	schema := []byte(`{
//	    "type": "object",
//	    "required": ["service"],
//	    "properties": {
//	        "service":   {"type": "string"},
//	        "port":      {"type": "integer", "default": 8080},
//	        "log_level": {"type": "string",  "default": "info",
//	                      "enum": ["debug","info","warn","error"]}
//	    }
//	}`)
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithJSONSchema(schema),
//	)
func WithJSONSchema(schema []byte) Option {
	return func(cfg *config) {
		// Compile eagerly so callers get a construction-time error for invalid
		// JSON or schema syntax, rather than discovering it on the first Load.
		if _, _, err := compileJSONSchema(schema); err != nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithJSONSchema", err))
			return
		}
		cfg.steps = append(cfg.steps, &schemaStep{
			selector: func(_ map[string]any) ([]byte, error) {
				return schema, nil
			},
		})
	}
}

// WithJSONSchemaFunc registers a dynamic schema resolver as a pipeline step.
// The selector is called during [Synthra.Load] with the current [*Values] and
// returns the JSON Schema bytes to use at that point in the pipeline. This
// enables the schema to be chosen based on a value already present in the
// config — for example an apiVersion field — without requiring a two-pass read.
//
// The bytes returned by the selector are compiled and the schema's "default"
// values are extracted and applied before validation runs, exactly as with
// [WithJSONSchema].
//
// Multiple [WithJSONSchemaFunc] calls are allowed; each adds an independent
// schema step that runs at the point it was registered. This is the mechanism
// for two-phase validation: register a pre-transform schema step to validate
// partial data, then register a post-transform schema step to validate the
// final form.
//
// If the selector returns an error, or the returned bytes fail to compile, or
// validation fails, Load returns a [*ConfigError] with Op [OpLoad] and Path
// "step[N]:schema" where N is the zero-based index of this step.
//
// The selector function must not be nil.
//
// Example — select schema version from the config's own apiVersion key:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("deployah.yaml"),
//	    synthra.WithJSONSchemaFunc(func(v *synthra.Values) ([]byte, error) {
//	        version := v.StringOr("apiVersion", "")
//	        if version == "" {
//	            return nil, errors.New("apiVersion is required")
//	        }
//	        return schema.GetManifestSchema(version)
//	    }),
//	)
//
// Example — two-phase validation (validate before and after substitution):
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("manifest.yaml"),
//	    synthra.WithJSONSchemaFunc(environmentsSchema), // validate raw environments
//	    synthra.WithEnvSubst(synthra.FromEnv()),         // substitute variables
//	    synthra.WithJSONSchemaFunc(manifestSchema),     // validate substituted manifest
//	)
func WithJSONSchemaFunc(selector func(*Values) ([]byte, error)) Option {
	return func(cfg *config) {
		if selector == nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithJSONSchemaFunc", errors.New("selector cannot be nil")))
			return
		}
		cfg.steps = append(cfg.steps, &schemaStep{
			selector: func(m map[string]any) ([]byte, error) {
				return selector(newValues(m))
			},
		})
	}
}

// WithValidator adds a custom validation function as a pipeline step. It runs
// a read-only check against the current [*Values] at the point in the pipeline
// where it was registered. Multiple validators run in registration order; the
// first error stops the pipeline.
//
// Panics inside the validator are recovered and reported as errors.
// The function must not be nil.
//
// If the validator returns an error or panics, Load returns a [*ConfigError]
// with Op [OpLoad] and Path "step[N]:validator" where N is the zero-based
// index of this step.
//
// Example:
//
//	cfg, err := synthra.New(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithValidator(func(v *synthra.Values) error {
//	        port := v.IntOr("port", 0)
//	        if port < 1 || port > 65535 {
//	            return fmt.Errorf("port %d out of range", port)
//	        }
//	        return nil
//	    }),
//	)
func WithValidator(fn func(*Values) error) Option {
	return func(cfg *config) {
		if fn == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithValidator", errors.New("validator cannot be nil")))
			return
		}
		cfg.steps = append(cfg.steps, &validatorStep{fn: func(m map[string]any) error {
			return fn(newValues(m))
		}})
	}
}

// WithTransform registers a function that transforms the configuration values
// as a pipeline step. The transform runs at the point in the pipeline where it
// was registered, after any preceding steps have completed.
//
// The function receives the current [*Values] and mutates it in place.
// Returning an error aborts Load with a [*ConfigError] whose Path identifies
// the failing step by its index and kind ("step[0]:transform",
// "step[1]:transform", ...).
//
// Multiple transforms are applied in registration order.
//
// Example — normalize log level to lowercase, then validate with a schema:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithTransform(func(v *synthra.Values) error {
//	        if level, err := v.String("log_level"); err == nil {
//	            return v.Set("log_level", strings.ToLower(level))
//	        }
//	        return nil
//	    }),
//	    synthra.WithJSONSchema(schema),
//	)
func WithTransform(fn func(*Values) error) Option {
	return func(cfg *config) {
		if fn == nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithTransform", errors.New("transform function cannot be nil")))
			return
		}
		cfg.steps = append(cfg.steps, &transformStep{
			fn: func(m map[string]any) (map[string]any, error) {
				v := newValues(m)
				if err := fn(v); err != nil {
					return nil, err
				}
				return v.m, nil
			},
		})
	}
}

// WithEnvSubst registers a transform that expands POSIX-style ${VAR}
// placeholders in all string values of the merged configuration map.
//
// The resolver argument must not be nil. To consult multiple sources,
// compose them with [Resolver.Or] — the first Resolver that reports found
// wins (highest priority first):
//
//	synthra.FromEnv().Prefix("APP_").Or(envFile).Or(synthra.FromMap(defaults))
//
// Supported syntax includes ${VAR}, ${VAR:-default}, ${VAR:=default},
// ${VAR^^}, ${VAR#pattern}, and more. The full set is documented at
// https://pkg.go.dev/github.com/fluxcd/pkg/envsubst.
//
// This is different from [WithEnv]. [WithEnv] is a source: it reads
// environment variables and adds them as configuration keys. For
// example, APP_SERVER_PORT=8080 becomes server.port in the config
// map. [WithEnvSubst] is a transform: it expands ${VAR} placeholders
// that appear inside string values already loaded from other sources.
// Both can be used together without overlap.
//
// Example — OS environment only:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(synthra.FromEnv()),
//	)
//
// Example — expand placeholders using a static map:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubst(synthra.FromMap(map[string]string{
//	        "ENV":  "production",
//	        "PORT": "8080",
//	    })),
//	)
//	// If config.yaml contains: host: "app-${ENV}.example.com"
//	// After Load: cfg.Get("host") => "app-production.example.com"
//
// Example — three-layer priority with Or (highest priority first):
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
//	// If DPY_VAR_PORT=9090 is set, port becomes "9090".
//	// If DPY_VAR_PORT is not set but PORT is in the .env file, that wins.
//	// If neither is set, the ${VAR:-default} fallback gives "3000".
func WithEnvSubst(r Resolver) Option {
	return func(cfg *config) {
		if r == nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithEnvSubst", errors.New("resolver cannot be nil")))
			return
		}
		cfg.steps = append(cfg.steps, &transformStep{fn: func(values map[string]any) (map[string]any, error) {
			v := newValues(values)
			if err := envsubstMap(v.m, r, ""); err != nil {
				return nil, fmt.Errorf("envsubst: %w", err)
			}
			return v.m, nil
		}})
	}
}

// WithEnvSubstFunc expands ${VAR} placeholders using a [Resolver] that is
// determined dynamically at Load time. The callback receives the current
// [*Values] and returns a Resolver (or an error that stops the pipeline).
//
// This follows the same pattern as [WithJSONSchemaFunc]: the Func suffix
// means "the input to this step is determined at Load time from the
// current values." Use this when the resolver depends on values that are
// only known after sources are merged — for example, a .env file path
// that is itself stored in the config file.
//
// The function must not be nil.
//
// Example — .env file path specified in config:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubstFunc(func(v *synthra.Values) (synthra.Resolver, error) {
//	        envPath := v.StringOr("envfile", "")
//	        if envPath == "" {
//	            return synthra.FromEnv(), nil
//	        }
//	        return synthra.FromEnvFile(envPath)
//	    }),
//	)
//
// Example — Vault resolver with setup that may fail:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithEnvSubstFunc(func(_ *synthra.Values) (synthra.Resolver, error) {
//	        client, err := vault.NewClient(vault.DefaultConfig())
//	        if err != nil {
//	            return nil, fmt.Errorf("vault client: %w", err)
//	        }
//	        return func(name string) (string, bool) {
//	            secret, err := client.Read("secret/data/" + name)
//	            if err != nil || secret == nil {
//	                return "", false
//	            }
//	            v, ok := secret.Data["value"].(string)
//	            return v, ok
//	        }, nil
//	    }),
//	)
func WithEnvSubstFunc(fn func(*Values) (Resolver, error)) Option {
	return func(cfg *config) {
		if fn == nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithEnvSubstFunc", errors.New("resolver function cannot be nil")))
			return
		}
		cfg.steps = append(cfg.steps, &transformStep{fn: func(values map[string]any) (map[string]any, error) {
			v := newValues(values)
			resolver, err := fn(v)
			if err != nil {
				return nil, fmt.Errorf("envsubst: %w", err)
			}
			err = envsubstMap(v.m, resolver, "")
			if err != nil {
				return nil, fmt.Errorf("envsubst: %w", err)
			}
			return v.m, nil
		}})
	}
}

// envsubstMap recursively walks values and expands ${VAR} placeholders
// in all string values using the mapping function. The prefix
// accumulates the dotted path for error messages.
func envsubstMap(values map[string]any, mapping func(string) (string, bool), prefix string) error {
	for k, v := range values {
		path := prefix + k
		switch val := v.(type) {
		case string:
			expanded, err := envsubst.Eval(val, mapping)
			if err != nil {
				return fmt.Errorf("key %q: %w", path, err)
			}
			values[k] = expanded
		case map[string]any:
			if err := envsubstMap(val, mapping, path+"."); err != nil {
				return err
			}
		case []any:
			if err := envsubstSlice(val, mapping, path); err != nil {
				return err
			}
		}
	}
	return nil
}

// envsubstSlice applies envsubst expansion to every element in a slice,
// recursing into nested maps and slices. The prefix is the parent key
// path; indices are appended as [N].
func envsubstSlice(slice []any, mapping func(string) (string, bool), prefix string) error {
	for i, elem := range slice {
		path := fmt.Sprintf("%s[%d]", prefix, i)
		switch val := elem.(type) {
		case string:
			expanded, err := envsubst.Eval(val, mapping)
			if err != nil {
				return fmt.Errorf("key %q: %w", path, err)
			}
			slice[i] = expanded
		case map[string]any:
			if err := envsubstMap(val, mapping, path+"."); err != nil {
				return err
			}
		case []any:
			if err := envsubstSlice(val, mapping, path); err != nil {
				return err
			}
		}
	}
	return nil
}
