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
	"io/fs"
	"os"
	"reflect"

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

// WithBinding returns an Option that configures the Synthra instance to
// bind configuration data to a struct. The target must be a non-nil pointer.
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
//	    synthra.WithBinding(&appCfg),
//	)
//	fmt.Println(appCfg.Server.Port) // populated from config
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

// WithJSONSchema adds a JSON Schema for validation and automatic default
// application. Synthra supports JSON Schema drafts 4, 6, 7, 2019-09, and
// 2020-12.
//
// # Validation
//
// The merged configuration map is validated against the schema during [Load],
// after schema defaults are applied and after any registered [WithTransform]
// functions have run. If validation fails, Load returns a [*ConfigError] with
// Op [OpLoad] and Path "json-schema".
//
// # Automatic defaults
//
// Synthra also extracts every "default" value declared in the schema and
// applies it to any key that is missing from the loaded configuration. This
// happens before transforms and validation run, so the schema validator always
// sees a fully populated map.
//
// Defaults are applied recursively at every level:
//   - "properties" — fills missing fixed-name keys in an object
//   - "patternProperties" — fills missing keys inside every existing map entry
//     whose name matches the regular-expression pattern
//   - "items" — fills missing keys inside each element of an array
//
// User-provided values are never overridden; only absent keys are filled.
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
//	                      "enum": ["debug","info","warn","error"]},
//	        "components": {
//	            "type": "object",
//	            "patternProperties": {
//	                "^[a-z0-9-]+$": {
//	                    "properties": {
//	                        "role":     {"type": "string",  "default": "service"},
//	                        "replicas": {"type": "integer", "default": 1}
//	                    }
//	                }
//	            }
//	        }
//	    }
//	}`)
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithJSONSchema(schema),
//	)
//	// If config.yaml contains only:
//	//   service: my-app
//	//   components:
//	//     web:
//	//       image: nginx
//	//
//	// After Load:
//	//   cfg.Get("port")                        => 8080      (schema default)
//	//   cfg.Get("log_level")                   => "info"    (schema default)
//	//   cfg.Get("components.web.role")         => "service" (patternProperties default)
//	//   cfg.Get("components.web.replicas")     => 1         (patternProperties default)
func WithJSONSchema(schema []byte) Option {
	return func(cfg *config) {
		compiled, raw, err := compileJSONSchema(schema)
		if err != nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithJSONSchema", err))
			return
		}
		cfg.jsonSchemaCompiled = compiled
		cfg.jsonSchemaRaw = raw
	}
}

// WithJSONSchemaSelector registers a lazy schema resolver that is called during
// [Synthra.Load] with the merged configuration values and returns the JSON Schema
// bytes to use for that load. This enables the schema to be chosen based on a
// value read from the config itself — for example an `apiVersion` field — without
// requiring a two-pass read.
//
// The selector runs after all sources are merged and before schema defaults,
// transforms, and validation. The bytes it returns are compiled and the schema's
// "default" values are extracted, so the full pipeline (defaults → transforms →
// validation) applies exactly as if [WithJSONSchema] had been used.
//
// [WithJSONSchema] and [WithJSONSchemaSelector] are mutually exclusive. Using both
// in the same [New] call is a construction-time error.
//
// Errors returned by the selector abort [Load] with a [*ConfigError] whose Op is
// [OpLoad] and Path is "json-schema-selector". Schema bytes that fail to compile
// produce the same error shape.
//
// Example — select schema version from the config's own apiVersion key:
//
//	cfg := synthra.MustNew(
//	    synthra.WithFile("deployah.yaml"),
//	    synthra.WithJSONSchemaSelector(func(values map[string]any) ([]byte, error) {
//	        version, ok := values["apiversion"].(string)
//	        if !ok || version == "" {
//	            return nil, errors.New("apiVersion is required")
//	        }
//	        return schema.GetManifestSchema(version)
//	    }),
//	)
//	err := cfg.Load(context.Background())
func WithJSONSchemaSelector(fn func(map[string]any) ([]byte, error)) Option {
	return func(cfg *config) {
		if fn == nil {
			cfg.validationErrors = append(cfg.validationErrors,
				NewConfigError(OpNew, "WithJSONSchemaSelector", errors.New("selector function cannot be nil")))
			return
		}
		cfg.jsonSchemaSelector = fn
	}
}

// WithValidator adds a custom validation function that runs against the
// merged configuration map after all sources are loaded. Multiple validators
// are executed in the order they are added; the first error stops evaluation.
// The function must not be nil.
//
// Example:
//
//	cfg, err := synthra.New(
//	    synthra.WithFile("config.yaml"),
//	    synthra.WithValidator(func(m map[string]any) error {
//	        port, _ := m["port"].(int)
//	        if port < 1 || port > 65535 {
//	            return fmt.Errorf("port %d out of range", port)
//	        }
//	        return nil
//	    }),
//	)
func WithValidator(fn func(map[string]any) error) Option {
	return func(cfg *config) {
		if fn == nil {
			cfg.validationErrors = append(cfg.validationErrors, NewConfigError(OpNew, "WithValidator", errors.New("validator cannot be nil")))
			return
		}
		cfg.customValidators = append(cfg.customValidators, fn)
	}
}
