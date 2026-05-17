# Dump merged configuration

Load from a YAML file and `APP_*` environment variables, then write the combined result to a new YAML file with `WithFileDumperAs` and `Dump`.

This is handy for debugging -- you can see exactly what Synthra resolved after merging all sources.

## Run

```bash
cd examples/dump
APP_SERVER_PORT=9090 go run .
cat effective-config.yaml
```

You can also pass a custom output path:

```bash
go run . /tmp/my-config.yaml
```

## Tests

```bash
cd examples/dump && go test -v
```

## Key ideas

1. **Inspect the merged state** -- `Dump` writes whatever `Load` produced.
2. **Any codec** -- `WithFileDumperAs` accepts `codec.YAML`, `codec.JSON`, or `codec.TOML`.
3. **No side effects on Load** -- the dumper only writes when you call `Dump` explicitly.
