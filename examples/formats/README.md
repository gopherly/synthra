# Explicit formats (`WithFileAs`)

Loads `app.json` as JSON, then merges `overrides.toml` as TOML. File extensions do not need to match the codec when you pass a decoder explicitly.

```bash
cd examples/formats && go run .
```

Tests:

```bash
cd examples/formats && go test -v
```
