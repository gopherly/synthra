# Basic Example

This example demonstrates the most basic usage of Synthra - loading configuration from a YAML file into a Go struct.

## Features Demonstrated

- **File Source**: Loading configuration from a YAML file
- **Struct Binding**: Mapping configuration to Go structs
- **Type Conversion**: Automatic conversion of different data types
- **Nested Structures**: Handling nested configuration objects
- **Arrays and Slices**: Loading string arrays and slices
- **Time Types**: Parsing time.Duration and time.Time values
- **URL Types**: Parsing URL strings into *url.URL

## Configuration Structure

The example includes various configuration types:

- **Basic Types**: string, int, bool, time.Duration
- **Complex Types**: time.Time, *url.URL
- **Collections**: []string (both as array and comma-separated string)
- **Nested Objects**: Worker configuration with timeout and address

## Running the Example

```bash
cd examples/basic
go run main.go
```

## Expected output

`fmt`’s default format for `time.Time` and `time.Duration` depends on locale and version, so do not rely on an exact string. After `go run .` you should see `Foo:bar`, `Timeout` as `10s`, `Debug:true`, worker address `http://localhost:8080`, and roles `admin` / `user`.

The struct maps the same YAML key `types` twice on purpose: **`Types`** (`[]string`) and **`Types2`** (`string`) both use `synthra:"types"` to show slice decoding versus leaving the raw comma-separated string.

## Tests

```bash
cd examples/basic && go test -v
```

The `config.yaml` file contains:

```yaml
foo: bar
timeout: 10s
debug: true
date: 2025-01-01T00:00:00+01:00
types: x1,x2,x3
roles:
  - admin
  - user
worker:
  timeout: 600
  address: http://localhost:8080
```

## Key Concepts

1. **Struct Tags**: Use `synthra:"field_name"` to map configuration keys to struct fields
2. **Type Safety**: Synthra automatically converts values to the appropriate Go types
3. **Nested Mapping**: Use dot notation in struct tags for nested configuration
4. **Multiple Formats**: Arrays can be loaded from YAML arrays or comma-separated strings
