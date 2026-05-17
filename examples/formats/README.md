# Explicit formats (`WithFileAs`)

Load `app.json` as JSON, then merge `overrides.toml` as TOML. When you pass the codec explicitly with `WithFileAs`, the file extension does not need to match.

## Run

```bash
cd examples/formats && go run .
```

Expected output: `app=formats-demo listen.port=4000 meta.region=local`

- `listen.port` starts as `3000` in the JSON file and is overridden to `4000` by the TOML file.
- `meta.region` only exists in the TOML file and is added to the merged result.

## Tests

```bash
cd examples/formats && go test -v
```

## Key ideas

1. **Mix formats freely** -- JSON, TOML, and YAML can all be combined in the same config.
2. **Explicit codec** -- `WithFileAs` tells Synthra how to decode the file instead of guessing from the extension.
3. **Same merge rules** -- later sources override earlier ones, regardless of format.
