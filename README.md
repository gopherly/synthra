# Synthra

[![CI](https://github.com/gopherly/synthra/actions/workflows/ci.yml/badge.svg)](https://github.com/gopherly/synthra/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/gopherly/synthra/branch/main/graph/badge.svg)](https://codecov.io/gh/gopherly/synthra)
[![Go Reference](https://pkg.go.dev/badge/gopherly.dev/synthra.svg)](https://pkg.go.dev/gopherly.dev/synthra)
[![Go Report Card](https://goreportcard.com/badge/gopherly.dev/synthra)](https://goreportcard.com/report/gopherly.dev/synthra)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](./LICENSE)
[![Slack](https://img.shields.io/badge/Slack-%23gopherly-4A154B?logo=slack)](https://slack.com/app_redirect?channel=C0B50TDEKN2)

**From many sources, one config.**

Synthra is a Go package that builds one configuration from many places. It reads from files, environment variables, Consul, in-memory bytes, and any custom source. It merges them in order, validates the result, and binds it to a struct if you want. The name comes from the Greek word *synthesis*, which means "to put together."

```bash
go get gopherly.dev/synthra
```

Requires Go 1.26 or later.

```go
import "gopherly.dev/synthra"
```

## How it works

```mermaid
flowchart LR
    S1[File] --> Merge
    S2[Env] --> Merge
    S3[Consul] --> Merge
    S4[Custom] --> Merge
    Merge --> P1

    subgraph Pipeline ["pipeline steps (run in registration order)"]
        direction LR
        P1["Step 0<br>schema / transform / validator"] --> P2
        P2["Step 1<br>..."] --> P3
        P3["Step N<br>..."]
    end

    P3 --> Bind["Bind to struct"]
    Bind --> Ready["Synthra ready"]
    Ready --> Read["Get / String / Int / ..."]
    Ready --> Dump["Dump"]

    style S1 fill:#dbeafe,stroke:#3b82f6,color:#1e3a5f
    style S2 fill:#dbeafe,stroke:#3b82f6,color:#1e3a5f
    style S3 fill:#dbeafe,stroke:#3b82f6,color:#1e3a5f
    style S4 fill:#dbeafe,stroke:#3b82f6,color:#1e3a5f
    style Merge fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style P1 fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style P2 fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style P3 fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style Pipeline fill:#fffbeb,stroke:#f59e0b,stroke-dasharray:5 5,color:#92400e
    style Bind fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style Ready fill:#d1fae5,stroke:#10b981,color:#064e3b
    style Read fill:#ede9fe,stroke:#8b5cf6,color:#3b0764
    style Dump fill:#ede9fe,stroke:#8b5cf6,color:#3b0764
```

## Why Synthra

Most Go services load configuration from more than one place. A YAML file holds the defaults, environment variables override them in production, and a key-value store like Consul holds shared settings. Synthra makes this simple:

- One small API for all sources.
- Clear merge order: later sources win over earlier ones.
- Twelve-Factor friendly: environment variables override files cleanly across environments.
- Automatic format detection from extension (`.yaml`, `.json`, `.toml`).
- JSON Schema defaults fill missing keys automatically.
- Pipeline processing: schema steps, transforms, and validators run in registration order for full flexibility.
- Dynamic schema selection with `WithJSONSchemaFunc` based on a value inside the config.
- Two-phase validation: validate before substitution and again after.
- Struct binding with type conversion, defaults, and validation.
- Case-insensitive keys with dot notation (`server.port`).
- Safe for concurrent use.
- Small core, optional extras. Consul is the only heavy dependency, and you only touch it if you need it.
- A `synthratest` helper package for tests.

## Contents

1. [How it works](#how-it-works)
2. [Quick start](#quick-start)
3. [Sources](#sources)
4. [Formats](#formats)
5. [Struct binding](#struct-binding)
6. [Default values](#default-values)
7. [JSON Schema defaults](#json-schema-defaults)
8. [Pipeline](#pipeline)
9. [Pipeline callbacks and Values](#pipeline-callbacks-and-values)
10. [Validation](#validation)
11. [Reading values](#reading-values)
12. [Merge order and precedence](#merge-order-and-precedence)
13. [Case insensitivity, casing, and dot notation](#case-insensitivity-casing-and-dot-notation)
14. [Environment variable naming](#environment-variable-naming)
15. [Dumping configuration](#dumping-configuration)
16. [Testing helpers](#testing-helpers)
17. [Custom sources and codecs](#custom-sources-and-codecs)
18. [Error handling](#error-handling)
19. [Thread safety](#thread-safety)
20. [Examples](#examples)
21. [License](#license)
22. [Contributing](#contributing)

## Quick start

Create a `config.yaml` file:

```yaml
server:
  host: "localhost"
  port: 8080
debug: true
```

Then load it:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "gopherly.dev/synthra"
)

type Config struct {
    Server struct {
        Host string `synthra:"host"`
        Port int    `synthra:"port"`
    } `synthra:"server"`
    Debug bool `synthra:"debug"`
}

func main() {
    var cfg Config

    s := synthra.MustNew(
        synthra.WithFile("config.yaml"),
        synthra.WithEnv("APP_"),
        synthra.WithBinding(&cfg),
    )

    if err := s.Load(context.Background()); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("listening on %s:%d (debug=%v)\n",
        cfg.Server.Host, cfg.Server.Port, cfg.Debug)
}
```

Set `APP_SERVER_PORT=9090` to override the YAML port at runtime.

## Sources

A source is any type whose `Load` method returns a `map[string]any` and an `error`. Synthra ships several built-in sources.

### File with automatic format detection

The format comes from the file extension. Supported extensions: `.yaml`, `.yml`, `.json`, `.toml`.

```go
synthra.WithFile("config.yaml")
synthra.WithFile("config.json")
synthra.WithFile("config.toml")
```

Paths support shell-style environment variable expansion: `${VAR}` or `$VAR`.

```go
synthra.WithFile("${CONFIG_DIR}/app.yaml")
```

### File with explicit format

Use this when the file has no extension, or when the extension does not match the real format.

```go
import "gopherly.dev/synthra/codec"

synthra.WithFileAs("config", codec.YAML)
synthra.WithFileAs("config.dat", codec.JSON)
```

### File inside an `io/fs.FS`

Useful for embedded files (`embed.FS`) and tests (`testing/fstest.MapFS`).

```go
import (
    "embed"
    "gopherly.dev/synthra"
)

//go:embed config.yaml
var configFS embed.FS

s := synthra.MustNew(
    synthra.WithFileFS(configFS, "config.yaml"),
)
```

You can also use `WithFileFSAs` to pass an explicit decoder.

### Environment variables

Pick a prefix. Synthra reads every variable with that prefix, removes it, lowercases the rest, and splits on `_` to build a nested map.

```go
synthra.WithEnv("APP_")
```

`APP_SERVER_PORT=8080` becomes `server.port = "8080"`.

See [Environment variable naming](#environment-variable-naming) for the full rules.

### In-memory content

Pass raw bytes and a decoder. Good for baked-in defaults.

```go
defaults := []byte(`
server:
  port: 3000
`)

synthra.WithContent(defaults, codec.YAML)
```

### Consul key-value store

Reads a key from Consul and decodes the value. The format is detected from the path, like for files.

```go
synthra.WithConsul("production/service.yaml")
```

`CONSUL_HTTP_ADDR` must be set. If it is missing, `New` returns an error at construction. For dev setups where Consul may not run, gate it with `WithIf`:

```go
synthra.WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "",
    synthra.WithConsul("production/service.yaml"),
)
```

This pattern does nothing when `CONSUL_HTTP_ADDR` is not set. The token, if any, comes from `CONSUL_HTTP_TOKEN`.

Use `WithConsulAs` when the path has no extension, or when the extension does not match the format:

```go
synthra.WithConsulAs("production/service", codec.JSON)
synthra.WithIf(os.Getenv("CONSUL_HTTP_ADDR") != "",
    synthra.WithConsulAs("production/service", codec.JSON),
)
```

### Custom source

Implement the `Source` interface and pass it through `WithSource`:

```go
type Source interface {
    Load(ctx context.Context) (map[string]any, error)
}

s := synthra.MustNew(
    synthra.WithSource(mySource),
)
```

The `source.NewMap` helper is useful for tests and embedded trees:

```go
import "gopherly.dev/synthra/source"

s := synthra.MustNew(
    synthra.WithSource(source.NewMap(map[string]any{
        "server": map[string]any{"port": 8080},
    })),
)
```

## Formats

The `codec` package ships ready-to-use codecs:

| Codec | Reads | Writes |
|-------|-------|--------|
| `codec.YAML` | yes | yes |
| `codec.JSON` | yes | yes |
| `codec.TOML` | yes | yes |
| `codec.EnvVar` | yes | no |

It also offers scalar decoders for single-value sources (for example, a Consul key that holds one number):

```go
codec.ParseInt("port")       // bytes -> map{"port": int(...)}
codec.ParseBool("debug")
codec.ParseString("name")
codec.ParseDuration("timeout")
codec.ParseTime("start")
codec.ParseAs("count", strconv.Atoi)  // generic parser
```

## Struct binding

Binding turns the merged map into a typed struct. Add `WithBinding` and pass a pointer to a struct.

```go
type Config struct {
    Host    string        `synthra:"host"`
    Port    int           `synthra:"port"`
    Timeout time.Duration `synthra:"timeout"`
    Roles   []string      `synthra:"roles"`
    URL     *url.URL      `synthra:"url"`
}

var cfg Config
s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithBinding(&cfg),
)

if err := s.Load(context.Background()); err != nil {
    log.Fatal(err)
}
```

The default struct tag is `synthra`. You can pick another tag name:

```go
synthra.WithTag("cfg")
```

Built-in type conversions:

- Strings, numbers, and booleans through the `cast` library.
- `time.Duration` from strings like `"30s"` or `"5m"`.
- `time.Time` from RFC 3339 strings (for example `"2025-01-01T00:00:00Z"`).
- `*url.URL` from any string URL.
- Slices from comma-separated strings or YAML/JSON arrays.

## Default values

Use the `default` struct tag for fallback values. A default applies only when the field stays at its zero value after binding.

```go
type Config struct {
    Host    string        `synthra:"host"    default:"localhost"`
    Port    int           `synthra:"port"    default:"8080"`
    Debug   bool          `synthra:"debug"   default:"false"`
    Timeout time.Duration `synthra:"timeout" default:"30s"`
}
```

You can also pass in defaults as a source. This is good when you want them visible in the merged map (for example, for `Dump`):

```go
defaults := []byte(`server: { port: 3000 }`)

synthra.MustNew(
    synthra.WithContent(defaults, codec.YAML),
    synthra.WithFile("config.yaml"),
    synthra.WithEnv("APP_"),
)
```

## JSON Schema defaults

When you pass a schema with `WithJSONSchema`, Synthra automatically extracts every `"default"` value declared in the schema and applies it to any key that is missing from the loaded configuration. This happens after sources are merged and before validation runs, so the schema validator always sees a fully populated map.

```go
schema := []byte(`{
    "type": "object",
    "properties": {
        "port":      {"type": "integer", "default": 8080},
        "log_level": {"type": "string",  "default": "info",
                      "enum": ["debug", "info", "warn", "error"]}
    }
}`)

s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithJSONSchema(schema),
)
// If config.yaml omits "port" and "log_level", they are set to 8080 and "info"
// before validation. Values present in config.yaml are never overridden.
```

Defaults are applied at every level of nesting, including `patternProperties`. For dynamic key names (like a map of named components), Synthra applies the `patternProperties` defaults to every existing key that matches the pattern:

```go
schema := []byte(`{
    "properties": {
        "components": {
            "patternProperties": {
                "^[a-z0-9-]+$": {
                    "properties": {
                        "role":     {"type": "string",  "default": "service"},
                        "replicas": {"type": "integer", "default": 1}
                    }
                }
            }
        }
    }
}`)

// config.yaml:
//   components:
//     web:
//       image: nginx
//     worker:
//       image: my-app
//       replicas: 3
//
// After Load:
//   components.web.role     => "service"  (from schema default)
//   components.web.replicas => 1          (from schema default)
//   components.worker.role  => "service"  (from schema default)
//   components.worker.replicas => 3       (from config.yaml, not overridden)
```

## Pipeline

After sources are merged, Synthra executes a pipeline of steps in the order they were registered. Each call to a pipeline option adds one step to the list. Steps run strictly in registration order; there is no implicit ordering between schema, transform, and validator steps.

Pipeline step options:

| Option | What it does |
|--------|-------------|
| `WithJSONSchema(bytes)` | Applies schema defaults then validates at this point in the pipeline. Schema bytes are validated at construction. |
| `WithJSONSchemaFunc(selector)` | Same as `WithJSONSchema` but schema bytes come from a callback that receives the current values. Use this when the schema depends on a value inside the config (e.g. `apiVersion`). |
| `WithTransform(fn)` | Applies an arbitrary map mutation at this point. |
| `WithEnvSubst(r)` | Expands `${VAR}` placeholders in all string values using a single `Resolver`. Compose multiple sources with `.Or`. Sugar over `WithTransform`. |
| `WithValidator(fn)` | Runs a read-only check. Does not modify values. |

Because steps are ordered, you can place a schema before a transform, a transform before a validator, or anything else you need.

### Dynamic schema selection

Use `WithJSONSchemaFunc` when the schema depends on a value inside the config itself. The most common case is an `apiVersion` field:

```go
cfg := synthra.MustNew(
    synthra.WithFile("manifest.yaml"),
    synthra.WithJSONSchemaFunc(func(v *synthra.Values) ([]byte, error) {
        version, err := v.String("apiVersion")
        if err != nil || version == "" {
            return nil, errors.New("apiVersion is required")
        }
        return schemaRegistry.Get(version)  // your own lookup
    }),
)
```

Multiple `WithJSONSchema` and `WithJSONSchemaFunc` calls are fully supported; each adds an independent schema step at the point it was registered.

### Two-phase validation

Register a schema step before substitution to validate raw fields, then register another after substitution to validate the final form:

```go
cfg := synthra.MustNew(
    synthra.WithFile("manifest.yaml"),
    // Step 1: validate the "environments" block on raw values.
    synthra.WithJSONSchemaFunc(func(_ *synthra.Values) ([]byte, error) {
        return environmentsSchema, nil
    }),
    // Step 2: expand ${VAR} placeholders from OS environment.
    synthra.WithEnvSubst(synthra.FromEnv()),
    // Step 3: validate the fully-substituted manifest.
    synthra.WithJSONSchemaFunc(func(_ *synthra.Values) ([]byte, error) {
        return manifestSchema, nil
    }),
)
```

See [`examples/multi-schema`](./examples/multi-schema) for a runnable demonstration.

### Transforms

`WithTransform` registers a function that processes the merged configuration map at the point it was registered. Multiple transforms run in registration order.

```go
s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithTransform(func(v *synthra.Values) error {
        if level := v.StringOr("logLevel", ""); level != "" {
            return v.Set("logLevel", strings.ToLower(level))
        }
        return nil
    }),
    synthra.WithJSONSchema(schema), // validates the normalized values
)
```

`WithEnvSubst` is a convenience transform that expands POSIX-style `${VAR}` placeholders in all string values. It supports defaults (`${VAR:-fallback}`), uppercase conversion (`${VAR^^}`), prefix stripping (`${VAR#prefix}`), and more.

Variable lookup is handled by a single [`Resolver`](https://pkg.go.dev/gopherly.dev/synthra#Resolver). Compose multiple sources with `.Or`. The first resolver that reports found wins (highest priority first):

```go
// OS environment only
s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithEnvSubst(synthra.FromEnv()),
)

// Static map
s = synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithJSONSchema(schema),
    synthra.WithEnvSubst(synthra.FromMap(map[string]string{
        "ENV":    "production",
        "REGION": "eu-west-1",
    })),
)
// If config.yaml has: envFile: ".env.${ENV}"
// After Load:         envFile => ".env.production"
```

Layer multiple resolvers for priority-based substitution using `.Or`. The first resolver to find a given variable name wins (highest priority first):

```go
envFile, err := synthra.FromEnvFile(".env")
if err != nil {
    log.Fatal(err)
}

s := synthra.MustNew(
    synthra.WithFile("deployah.yaml"),
    synthra.WithEnvSubst(
        synthra.FromEnv().Prefix("DPY_VAR_").  // highest: prefixed OS env
            Or(envFile).                         // middle:  .env file
            Or(synthra.FromMap(manifestVars)),   // lowest:  static defaults
    ),
)
// config.yaml: port: ${PORT:-3000}
// If DPY_VAR_PORT=9090 is set, port becomes "9090".
// If DPY_VAR_PORT is not set but PORT is in the .env file, that value is used.
// If neither is set, the ${VAR:-default} fallback provides "3000".
```

Each variable lookup walks the chain from left to right and stops at the first resolver that reports found:

```mermaid
flowchart LR
    Var["${PORT:-3000}"] --> R1
    Out["resolved value"]

    subgraph Chain ["resolver chain (first match wins)"]
        direction LR
        R1["FromEnv().Prefix(DPY_VAR_)"] -->|"not found"| R2
        R2["FromEnvFile(.env)"] -->|"not found"| R3
        R3["FromMap(defaults)"] -->|"not found"| Fallback["inline default"]
    end

    R1 -->|found| Out
    R2 -->|found| Out
    R3 -->|found| Out
    Fallback --> Out

    style Var fill:#dbeafe,stroke:#3b82f6,color:#1e3a5f
    style R1 fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style R2 fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style R3 fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style Chain fill:#fffbeb,stroke:#f59e0b,stroke-dasharray:5 5,color:#92400e
    style Fallback fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style Out fill:#d1fae5,stroke:#10b981,color:#064e3b
```

The available resolver constructors are:

- `synthra.FromMap(m)`: looks up variables from a `map[string]string`
- `synthra.FromEnv()`: looks up variables using `os.LookupEnv` (reads live env at Load time)
- `synthra.FromEnvFile(path)`: parses a `.env` file eagerly and returns a map-backed resolver; returns an error if the file is missing or malformed
- `.Prefix(prefix)`: method on any `Resolver` that prepends a namespace prefix to each lookup (e.g. `FromEnv().Prefix("APP_")` resolves `PORT` from `APP_PORT`)
- `.Or(fallbacks...)`: method on any `Resolver` that chains fallbacks with first-match-wins semantics

Load a `.env` file and combine it with OS env (OS env wins via first match):

```go
envFile, err := synthra.FromEnvFile(".env")
if err != nil {
    log.Fatal(err)
}

s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithEnvSubst(synthra.FromEnv().Or(envFile)),
)
```

Use `WithEnvSubstFunc` when the resolver depends on values already loaded from sources. For example, a `.env` file path stored inside the config file:

```go
s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithEnvSubstFunc(func(v *synthra.Values) (synthra.Resolver, error) {
        envPath := v.StringOr("envfile", "")
        if envPath == "" {
            return synthra.FromEnv(), nil
        }
        envFile, err := synthra.FromEnvFile(envPath)
        if err != nil {
            return nil, err
        }
        // OS env takes priority over the .env file.
        return synthra.FromEnv().Or(envFile), nil
    }),
)
```

**`WithEnv` and `WithEnvSubst` solve different problems:**

- `WithEnv` is a source. It reads environment variables and adds them to the config map. For example, `APP_SERVER_PORT=8080` becomes `server.port`. Use this when you want env vars to be config keys.
- `WithEnvSubst` is a transform. It expands `${VAR}` placeholders that are already present in string values loaded from files or other sources. Use this when your config files contain placeholder strings that should be filled from the environment or a map.

Both can be used together. They do not overlap.

`WithEnvSubst` also works with `patternProperties` defaults. If the schema default for a field is `".env.${NAME}"`, the substitution runs after the default is applied and fills in the placeholder.

## Pipeline callbacks and Values

`WithTransform`, `WithValidator`, `WithEnvSubstFunc`, and `WithJSONSchemaFunc` receive a `*synthra.Values` wrapper instead of a raw `map[string]any`. The wrapper gives you safe, case-insensitive, typed access.

### Why *Values

Direct map access is case-sensitive at the Go language level. If your YAML says `apiVersion` but the merged map ends up storing it under a slightly different casing, `values["apiVersion"]` returns nil. The `*Values` wrapper fixes that. All its methods are case-insensitive.

### Reading

```go
v.Get("metadata.name")         // any, case-insensitive
v.Has("server.tls.enabled")    // bool
v.String("apiVersion")         // (string, error)
v.IntOr("server.port", 8080)   // int with default
```

### Writing

```go
_ = v.Set("metadata.region", "eu-west-1") // creates intermediate maps
v.Delete("debug.experimental")
```

### Walking the tree

```go
v.Walk(func(path string, val any) (any, bool) {
    if s, ok := val.(string); ok && strings.HasPrefix(s, "${") {
        return strings.TrimPrefix(s, "${"), true
    }
    return val, false
})
```

### Escape hatch

When you must hand the underlying map to code that expects a plain `map[string]any`, call `v.Raw()`. Mutations on the returned map are visible through the same `*Values`.

### Map stage vs binding stage

`WithTransform`, `WithValidator`, `WithEnvSubstFunc`, and `WithJSONSchemaFunc` run at the map stage, before binding, on the merged `*Values`.

`OnBound[T]` is a binding-scoped option that goes inside `WithBinding[T]`. It runs at the binding stage: after the bound struct is decoded and defaults applied, but before its `Validate()` method (if it implements the `Validator` interface). The type parameter is inferred from the closure, and the compiler enforces that it matches the binding target.

```go
synthra.WithFile("config.yaml"),
synthra.WithTransform(func(v *synthra.Values) error {
    if v.StringOr("env", "dev") == "prod" {
        return v.Set("logging.level", "warn")
    }
    return nil
}),
synthra.WithBinding(&app,
    synthra.OnBound(func(a *App) error {
        a.Logging.Level = strings.ToLower(a.Logging.Level)
        return nil
    }),
),
```

### Compile-time safety for OnBound

Because `OnBound[T]` is a sub-option of `WithBinding[T]`, Go infers the same `T` for both. If the closure type does not match the binding target, you get a compile error, not a runtime panic:

```go
var server Server
synthra.WithBinding(&server,
    synthra.OnBound(func(a *App) error { ... }),  // compile error
)
```

Here is the full sequence when `Load` runs:

```mermaid
flowchart TB
    Sources["sources merged (last wins)"] --> MapStage

    subgraph MapStage ["map stage"]
        direction TB
        Steps["pipeline steps in order<br>schema / transform / envsubst / validator"]
    end

    MapStage --> Bind["bind into struct<br>mapstructure + default tags"]
    Bind --> BindStage

    subgraph BindStage ["binding stage"]
        direction TB
        Hooks["OnBound[T] hooks (in order)"] --> Val["Validate() if implemented"]
    end

    BindStage --> Ready["ready"]

    style Sources fill:#dbeafe,stroke:#3b82f6,color:#1e3a5f
    style Steps fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style MapStage fill:#fffbeb,stroke:#f59e0b,stroke-dasharray:5 5,color:#92400e
    style Bind fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style Hooks fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style Val fill:#fef3c7,stroke:#f59e0b,color:#78350f
    style BindStage fill:#fffbeb,stroke:#f59e0b,stroke-dasharray:5 5,color:#92400e
    style Ready fill:#d1fae5,stroke:#10b981,color:#064e3b
```

## Validation

Synthra supports three ways to validate, and you can combine them.

### 1. Validator interface on the bound struct

Add a `Validate() error` method on your struct. Synthra calls it after binding.

```go
type Config struct {
    Port int `synthra:"port"`
}

func (c *Config) Validate() error {
    if c.Port < 1 || c.Port > 65535 {
        return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
    }
    return nil
}
```

### 2. JSON Schema

Pass a schema as raw bytes. Synthra fills in `"default"` values, then validates the merged map before binding.

```go
schema := []byte(`{
    "type": "object",
    "required": ["service", "port"],
    "properties": {
        "service": {"type": "string", "minLength": 1},
        "port":    {"type": "integer", "minimum": 1, "maximum": 65535, "default": 8080}
    }
}`)

s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithJSONSchema(schema),
)
```

Synthra supports JSON Schema Draft 4, Draft 6, Draft 7, Draft 2019-09, and Draft 2020-12. See [JSON Schema defaults](#json-schema-defaults) for details on default application.

### 3. Custom validator function

Use `WithValidator` for cross-field rules or any logic that does not fit a schema.

```go
synthra.WithValidator(func(v *synthra.Values) error {
    if !v.Has("server.tls") {
        return nil
    }
    if enabled := v.BoolOr("server.tls.enabled", false); enabled {
        if !v.Has("server.tls.cert") || !v.Has("server.tls.key") {
            return errors.New("tls.cert and tls.key are required when tls.enabled is true")
        }
    }
    return nil
})
```

You can add more than one validator. Synthra runs them in order. The first error stops `Load`.

## Reading values

After `Load`, you have several ways to read values.

### Bound struct (preferred for typed code)

If you used `WithBinding`, just use the struct.

```go
fmt.Println(cfg.Server.Host, cfg.Server.Port)
```

### Strict typed methods

These return an error when the key is missing or the value cannot be converted.

```go
port, err := s.Int("server.port")
host, err := s.String("server.host")
debug, err := s.Bool("debug")
rate, err := s.Float64("rate")
timeout, err := s.Duration("timeout")
when, err := s.Time("start_time")
tags, err := s.StringSlice("tags")
ports, err := s.IntSlice("ports")
meta, err := s.StringMap("metadata")
```

Use `errors.Is(err, synthra.ErrKeyNotFound)` to check for a missing key.

### "Or" methods with a default

These never return an error. They return the default when the key is missing or cannot be converted.

```go
host := s.StringOr("server.host", "localhost")
port := s.IntOr("server.port", 8080)
debug := s.BoolOr("debug", false)
timeout := s.DurationOr("timeout", 30*time.Second)
tags := s.StringSliceOr("tags", []string{"default"})
```

Other Or methods exist for `Int64`, `Float64`, `Time`, `IntSlice`, and `StringMap`. See the [API docs](https://pkg.go.dev/gopherly.dev/synthra) for the full list.

### Generic `Get` and `GetOr`

For type-safe access with one function, use the generic helpers:

```go
port, err := synthra.Get[int](s, "server.port")
host := synthra.GetOr(s, "server.host", "localhost")
```

The type comes from the type parameter, or from the default value.

### Raw access

`Get(key)` returns the value as `any`. It returns `nil` when the key is missing.

```go
v := s.Get("server.port")  // any
```

`Values()` returns a copy of the merged map. Treat it as read-only.

```go
all := s.Values() // *map[string]any
```

## Merge order and precedence

Sources are merged in the order you add them. Later sources override earlier ones. Nested maps merge by key. Other values (strings, numbers, slices) are replaced as a whole.

```go
synthra.MustNew(
    synthra.WithContent(defaults, codec.YAML),   // 1. baked-in defaults
    synthra.WithFile("config.yaml"),             // 2. file on disk
    synthra.WithFile("override.json"),           // 3. another file
    synthra.WithEnv("APP_"),                     // 4. environment (wins)
)
```

In this example, environment variables have the highest precedence.

## Case insensitivity, casing, and dot notation

Synthra keeps the casing your config sources use. If your file says `apiVersion`, the loaded map will have `apiVersion` too. Only the matching is case-insensitive: you can read the same key as `apiVersion` or `APIVERSION` and get the same value.

```go
s.Int("server.port")  // works
s.Int("Server.Port")  // also works
s.Int("SERVER.PORT")  // also works
```

Keys use dot notation: `server.port` walks into `server` and reads `port`.

### The one exception: environment variables

Environment variables are uppercase by convention (`APP_API_VERSION`), so the env source lowercases them to produce a nested map. A `WithEnv("APP_")` source always contributes lowercase keys like `apiversion`. When env meets another source that already has the same key in a different casing, the case-insensitive merge keeps the first source's casing and overrides only the value. So if your YAML says `apiVersion: v1` and `APP_APIVERSION=v2` is set, the final map has `apiVersion: v2`.

### Two sources with different casing

When two non-env sources use different casings for the same key, the first source wins for the name and the last source wins for the value. So if `base.yaml` has `ApiVersion: v1` and `override.yaml` has `apiVersion: v2`, the final map looks like `ApiVersion: v2`. The typo in the base file is preserved.

To avoid that, register a JSON Schema. Before validation runs, Synthra renames any case-different keys in the data to match the schema's `"properties"` declarations. So `ApiVersion: v2` becomes `apiVersion: v2` if your schema says `"properties": {"apiVersion": ...}`.

```text
# base.yaml -> ApiVersion: v1
# override.yaml -> apiVersion: v2
# result: ApiVersion: v2  (first writer's casing wins)

# Same files, with a schema declaring apiVersion:
# result: apiVersion: v2  (schema is the authority)

# config.yaml -> apiVersion: v1
# APP_APIVERSION=v2
# result: apiVersion: v2  (YAML casing wins, env overrides value)
```

`patternProperties` and `additionalProperties` dynamic keys are not renamed by the schema. Keys inside list elements are only renamed when the schema declares an `items` object for that list.

## Environment variable naming

Given prefix `APP_`:

1. The prefix is removed.
2. The rest is lowercased.
3. Underscores split into nested keys.

| Variable | Key |
|----------|-----|
| `APP_PORT=8080` | `port` |
| `APP_SERVER_HOST=db` | `server.host` |
| `APP_DATABASE_PRIMARY_HOST=db` | `database.primary.host` |
| `APP_TAGS=a,b,c` | `tags` (string, splits to slice on read) |

A field like `server.read.timeout` maps to `APP_SERVER_READ_TIMEOUT` when the prefix is `APP_`.

## Dumping configuration

Synthra can write the merged configuration to a file. The format comes from the file extension, just like for sources.

```go
s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithEnv("APP_"),
    synthra.WithFileDumper("effective.yaml"),  // format from extension
)

s.Load(context.Background())
s.Dump(context.Background())  // writes effective.yaml
```

For an explicit format:

```go
synthra.WithFileDumperAs("output", codec.JSON)
```

You can also write your own dumper by implementing the `Dumper` interface and passing it with `WithDumper`.

```go
type Dumper interface {
    Dump(ctx context.Context, values *map[string]any) error
}
```

## Testing helpers

The `synthratest` package provides helpers for tests.

```go
import (
    "testing"

    "github.com/stretchr/testify/require"
    "gopherly.dev/synthra"
    "gopherly.dev/synthra/source"
    "gopherly.dev/synthra/synthratest"
)

func TestServer(t *testing.T) {
    cfg := synthratest.Load(t, map[string]any{
        "server": map[string]any{"port": 8080, "host": "127.0.0.1"},
    })

    port, err := cfg.Int("server.port")
    require.NoError(t, err)
    require.Equal(t, 8080, port)
}
```

Highlights:

- `synthratest.Config(t, opts...)`: build a `*Synthra` without calling `Load`.
- `synthratest.Load(t, map, opts...)`: build and load with a map source.
- `synthratest.LoadFile(t, format, content)`: write a temp file and load it.
- `synthratest.WriteFile(t, format, content)`: write a temp config file and return its path.
- `synthratest.Dumper`: a recording dumper for tests.
- `synthratest.FuncCodec`: a codec test double with function fields for `Decode` and `Encode`.
- `synthratest.ErrSource(err)`: a source that always returns the given error.
- `synthratest.AssertString`, `AssertInt`, `AssertBool`, `AssertStringSlice`, `AssertDumped`: shortcut assertions.

## Custom sources and codecs

### Custom source

Implement `Source`:

```go
type vaultSource struct {
    path string
}

func (s *vaultSource) Load(ctx context.Context) (map[string]any, error) {
    // fetch from your secret store
    return map[string]any{
        "db": map[string]any{
            "password": "from-vault",
        },
    }, nil
}

synthra.WithSource(&vaultSource{path: "secret/data/db"})
```

### Custom codec

Implement `codec.Codec` (or `codec.Decoder` only if you do not need to dump):

```go
type myCodec struct{}

func (myCodec) Decode(data []byte, v any) error { /* ... */ }
func (myCodec) Encode(v any) ([]byte, error)   { /* ... */ }

synthra.WithFileAs("config.custom", myCodec{})
```

## Error handling

Synthra returns structured errors of type `*ConfigError`. They follow the shape of `os.PathError`:

```go
type ConfigError struct {
    Op   string  // "new", "load", "dump", or "get"
    Path string  // where the error happened (source index, field, step index, ...)
    Err  error   // the underlying cause
}
```

Pipeline step errors use the path format `"step[N]:kind"` where N is the zero-based step index and kind is `"schema"`, `"transform"`, or `"validator"`. For example, `"step[0]:schema"` means the first registered schema step failed.

Use `errors.As` to read the operation:

```go
if err := s.Load(ctx); err != nil {
    var ce *synthra.ConfigError
    if errors.As(err, &ce) {
        log.Error("load failed", "op", ce.Op, "path", ce.Path, "err", ce.Err)
    }
    return err
}
```

Use `errors.Is` for fixed reasons:

```go
_, err := s.Int("server.port")
if errors.Is(err, synthra.ErrKeyNotFound) {
    return useDefaultPort()
}
```

Sentinel errors:

- `synthra.ErrNilConfig`: a typed accessor was called on a nil `*Synthra`.
- `synthra.ErrKeyNotFound`: the key is missing for a strict accessor.
- `synthra.ErrNilContext`: `Load` or `Dump` got a nil context.

`New` can return a joined error when more than one option is invalid. Use `errors.As` on it the same way.

## Thread safety

A `*Synthra` is safe for use by many goroutines:

- `Load` can be called many times. The internal map is replaced atomically when loading succeeds.
- All read methods (`Get`, `String`, `Int`, `Values`, ...) hold a read lock.
- `Dump` reads a snapshot of the current values, so dumpers do not block reads.
- The bound struct is not protected. If you re-load while another goroutine reads the struct, you are responsible for synchronizing access yourself.

## Examples

The [`examples/`](./examples) folder has small, runnable programs. Each one has its own README and tests.

| Folder | Topic |
|--------|-------|
| [`basic`](./examples/basic) | YAML file and struct binding |
| [`webapp`](./examples/webapp) | YAML defaults plus `WEBAPP_*` overrides, binding, and `Validate` |
| [`testing`](./examples/testing) | `synthratest.Config` and `source.NewMap` |
| [`schema`](./examples/schema) | `WithJSONSchema` defaults and `patternProperties` |
| [`casing`](./examples/casing) | Case-insensitive merge and schema as casing authority |
| [`hooks`](./examples/hooks) | `WithTransform`, `WithValidator`, and `OnBound[T]` in one pipeline |
| [`codecs`](./examples/codecs) | `WithFileAs` (JSON, TOML) and `WithFileDumperAs` (YAML dump) |
| [`envsubst-layered`](./examples/envsubst-layered) | Three-layer `Resolver.Or` precedence |
| [`multi-schema`](./examples/multi-schema) | Two-phase validation with `WithJSONSchemaFunc` |
| [`consul`](./examples/consul) | Optional Consul source via `WithIf` |

Run them all with:

```bash
go test ./examples/...
```

## License

Synthra is released under the [Apache License 2.0](./LICENSE).

## Community

Join [#gopherly](https://slack.com/app_redirect?channel=C0B50TDEKN2) on the [Gophers Slack](https://invite.slack.golangbridge.org/) for discussion and updates.

## Contributing

Contributions are welcome. Please open an issue first to discuss larger changes before sending a pull request.

This project uses Nix for development. Run `nix develop` to enter the shell, then:

- `nix run .#lint` to run the linter and check formatting.
- `nix run .#fmt` to auto-fix formatting.
- `nix run .#test-unit` to run unit tests.
