# JSON Schema validation

Loads `config.yaml` and validates the merged map against `schema.json` before the configuration is committed.

Run from this directory:

```bash
cd examples/jsonschema && go run .
```

Tests:

```bash
cd examples/jsonschema && go test -v
```
