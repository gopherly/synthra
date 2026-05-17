# Embedded defaults (`WithContent`)

Bake default values into your binary, override them with a file, then let environment variables win last.

Sources are merged in order -- later values replace earlier ones:

1. `WithContent` -- small YAML defaults embedded in Go code
2. `WithFile("overrides.yaml")` -- checked-in overrides
3. `WithEnv("DEMO_")` -- highest precedence

## Run

```bash
cd examples/defaults
DEMO_SERVER_PORT=9999 go run .
```

Expected output: `server.name=from-file server.port=9999`

- `server.name` comes from `overrides.yaml` (replaced the baked-in default)
- `server.port` comes from the `DEMO_SERVER_PORT` env var (replaced the file value of `7000`)

## Tests

```bash
cd examples/defaults && go test -v
```

## Key ideas

1. **Merge order matters** -- sources added later override earlier ones on the same key.
2. **Embedded defaults** -- `WithContent` takes a `[]byte` and a codec, so you can use `go:embed` or a literal.
3. **Three layers** -- defaults, file, environment is a common production pattern.
