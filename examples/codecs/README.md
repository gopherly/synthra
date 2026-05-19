# Codecs example

Shows how to use explicit codecs for both reading and writing configuration.

## What it shows

- `WithFileAs(path, codec.JSON)`: load a JSON file explicitly (no extension guessing)
- `WithFileAs(path, codec.TOML)`: load a TOML override on top of it
- `WithFileDumperAs(path, codec.YAML)`: dump the merged effective configuration to a YAML file

## Files

- `app.json`: base configuration in JSON format
- `overrides.toml`: overrides in TOML format; `listen.port` here wins over the JSON value
- `effective-config.yaml`: written on `Dump` (not committed)

## Run

```bash
cd examples/codecs && go run .
```

Output: `app=formats-demo listen.port=4000 meta.region=local`

A file `effective-config.yaml` is also written with the fully merged state.

## Tests

```bash
cd examples/codecs && go test -v
```

## Key ideas

1. `WithFileAs` lets you load any format regardless of file extension.
2. Source order is priority order. Later sources win on conflicting keys.
3. `WithFileDumperAs` writes the fully merged map, not the last-loaded file. Useful for debugging or snapshotting effective configuration.
