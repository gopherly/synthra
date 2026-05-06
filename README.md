# Synthra

**From many sources, one state.**

Synthra is a Go library that builds one configuration from many places. It reads from files, environment variables, Consul, in-memory bytes, and any custom source. It merges them in order, validates the result, and binds it to a struct if you want. The name comes from σύνθεσις (*synthesis*), which means "to put together into a whole".

```bash
go get gopherly.dev/synthra
```

API documentation: [pkg.go.dev/gopherly.dev/synthra](https://pkg.go.dev/gopherly.dev/synthra)

## Why Synthra

Most Go services load configuration from more than one place. A YAML file holds the defaults, environment variables override them in production, and a key-value store like Consul holds shared settings. Synthra makes this simple:

- One small API for all sources.
- Clear merge order: later sources win over earlier ones.
- Format detection from the file extension (`.yaml`, `.yml`, `.json`, `.toml`).
- Struct binding with type conversion, default values, and validation.
- Case-insensitive keys with dot notation (`server.port`).
- Safe for use from many goroutines at the same time.
- Small core, optional extras. The Consul source is the only heavy dependency, and it is only loaded if you import it through the option.
- A `synthratest` helper package for tests.

## Contents

1. [Quick start](#quick-start)
2. [Sources](#sources)
3. [Formats](#formats)
4. [Struct binding](#struct-binding)
5. [Default values](#default-values)
6. [Validation](#validation)
7. [Reading values](#reading-values)
8. [Merge order and precedence](#merge-order-and-precedence)
9. [Case insensitivity and dot notation](#case-insensitivity-and-dot-notation)
10. [Environment variable naming](#environment-variable-naming)
11. [Dumping configuration](#dumping-configuration)
12. [Testing helpers](#testing-helpers)
13. [Custom sources and codecs](#custom-sources-and-codecs)
14. [Error handling](#error-handling)
15. [Thread safety](#thread-safety)
16. [Examples](#examples)
17. [License](#license)

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

A *source* is anything that returns a `map[string]any`. Synthra ships several built-in sources.

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

`CONSUL_HTTP_ADDR` must be set. If it is missing, `New` returns an error at construction. For dev setups where Consul may not run, use the optional variant:

```go
synthra.WithConsulOptional("production/service.yaml")
```

This option does nothing when `CONSUL_HTTP_ADDR` is not set. The token, if any, comes from `CONSUL_HTTP_TOKEN`.

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

Pass a schema as raw bytes. Synthra validates the merged map before binding.

```go
schema := []byte(`{
    "type": "object",
    "required": ["service", "port"],
    "properties": {
        "service": {"type": "string", "minLength": 1},
        "port":    {"type": "integer", "minimum": 1, "maximum": 65535}
    }
}`)

s := synthra.MustNew(
    synthra.WithFile("config.yaml"),
    synthra.WithJSONSchema(schema),
)
```

### 3. Custom validator function

Use `WithValidator` for cross-field rules or any logic that does not fit a schema.

```go
synthra.WithValidator(func(m map[string]any) error {
    server, _ := m["server"].(map[string]any)
    tls, _ := server["tls"].(map[string]any)
    if enabled, _ := tls["enabled"].(bool); enabled {
        if tls["cert"] == nil || tls["key"] == nil {
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

### Generic `Get` and `GetOr`

For type-safe access with one function, use the generic helpers:

```go
port, err := synthra.Get[int](s, "server.port")
host := synthra.GetOr(s, "server.host", "localhost")
```

The type comes from the type parameter, or from the default value.

### Raw access

`Get(key)` returns `any` and `nil` when the key is missing.

```go
v := s.Get("server.port")  // any
```

`Values()` returns a shallow copy of the merged map. Treat it as read-only.

```go
all := s.Values()
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

## Case insensitivity and dot notation

Synthra lowercases every key when it merges sources. Reads are also case-insensitive.

```go
s.Int("server.port")  // works
s.Int("Server.Port")  // also works
s.Int("SERVER.PORT")  // also works
```

Keys use dot notation: `server.port` walks into `server` and reads `port`. A key that contains a dot literal is not supported. If you store such keys, read them through `Values()`.

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
- `synthratest.Dumper`: a recording dumper for tests.
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
    Path string  // diagnostic locator (source index, field, schema name, ...)
    Err  error   // the underlying cause
}
```

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
- The bound struct is not protected. If you re-load while another goroutine reads the struct, you need your own synchronization.

## Examples

The [`examples/`](./examples) folder has small, runnable programs. Each one has its own README and tests.

| Folder | Topic |
|--------|-------|
| [`basic`](./examples/basic) | YAML file and struct binding |
| [`environment`](./examples/environment) | Environment variables only |
| [`webapp`](./examples/webapp) | YAML defaults plus `WEBAPP_*` overrides, binding, and `Validate` |
| [`formats`](./examples/formats) | JSON and TOML with explicit codecs |
| [`defaults`](./examples/defaults) | Baked-in defaults, then file, then env |
| [`jsonschema`](./examples/jsonschema) | JSON Schema validation |
| [`customvalidator`](./examples/customvalidator) | Cross-field check with `WithValidator` |
| [`dump`](./examples/dump) | Writing the merged state to a file |
| [`consul`](./examples/consul) | Optional Consul source for dev and prod |
| [`testing`](./examples/testing) | Using `synthratest` and `source.NewMap` |

Run them all with:

```bash
go test ./examples/...
```

## License

Synthra is released under the [Apache License 2.0](./LICENSE).
