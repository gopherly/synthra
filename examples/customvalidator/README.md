# Custom validator (`WithValidator`)

Run your own checks on the merged configuration map. This example enforces a rule: when `server.tls.enabled` is true, both `server.tls.cert.file` and `server.tls.key.file` must be set.

## What it shows

- `WithValidator` accepts a `func(map[string]any) error`
- The function receives the full merged config as a nested map
- Returning a non-nil error makes `Load` fail

## How it works

Synthra calls your validator after all sources are merged but before the result is committed. The map you receive looks like the YAML structure -- nested `map[string]any` values that you walk manually.

```go
cfg, err := synthra.New(
    synthra.WithFile(path),
    synthra.WithValidator(tlsPathsConsistent),
)
```

The validator in this example (`tlsPathsConsistent`) digs into `server.tls`, checks if `enabled` is true, and returns an error when the cert or key path is empty.

## Run

Valid config (TLS enabled with both paths set):

```bash
cd examples/customvalidator && go run .
```

Invalid config (TLS enabled but paths are empty):

```bash
cd examples/customvalidator && go run . config-invalid.yaml
```

## Tests

```bash
cd examples/customvalidator && go test -v
```

## Key ideas

1. **Cross-field rules** -- unlike JSON Schema (which checks types and structure), a validator can enforce relationships between fields.
2. **Plain function** -- no interface to implement. Write a function, pass it in.
3. **Runs on the merged map** -- all sources (files, env vars, etc.) are already combined when your function runs.
