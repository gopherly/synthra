# Basic example

Load a YAML file into a Go struct. This is the simplest way to use Synthra.

## What it shows

- Loading configuration from a YAML file with `WithFile`
- Binding values to a Go struct with `WithBinding`
- Automatic type conversion for `time.Duration`, `time.Time`, `*url.URL`, `bool`, and `string`
- Nested structs (the `Worker` field)
- Slices from YAML arrays (`roles`) and comma-separated strings (`types`)

## Run

```bash
cd examples/basic && go run .
```

## Expected output

You should see `Foo:bar`, `Timeout` as `10s`, `Debug:true`, a worker address of `http://localhost:8080`, and roles `admin` / `user`.

The struct maps the same YAML key `types` twice on purpose: `Types` (`[]string`) and `Types2` (`string`) both use `synthra:"types"` to show how Synthra decodes the same value into a slice versus keeping the raw comma-separated string.

## Tests

```bash
cd examples/basic && go test -v
```

## The config file

`config.yaml` contains:

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

## Key ideas

1. **Struct tags** -- use `synthra:"key"` to map a YAML key to a struct field.
2. **Nesting** -- embed a struct field and give it a tag; Synthra walks into the matching YAML object automatically.
3. **Type safety** -- values are converted to the field's Go type at load time. A mismatch is an error, not a silent zero.
4. **Slices** -- YAML arrays and comma-separated strings both work.
