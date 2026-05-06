# Dump merged configuration

Loads `config.yaml`, applies `APP_*` environment overrides, then writes the **merged** map to YAML via `WithFileDumperAs` and `Dump`.

```bash
cd examples/dump
APP_SERVER_PORT=9090 go run .
cat effective-config.yaml
```

Tests:

```bash
cd examples/dump && go test -v
```
