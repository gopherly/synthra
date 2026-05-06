# Synthra examples

Runnable programs under this directory complement the shorter snippets in [example_test.go](../example_test.go) and the package overview on [pkg.go.dev](https://pkg.go.dev/gopherly.dev/synthra).

## Progression

| Directory | What it shows |
|-----------|----------------|
| [basic](./basic/) | YAML file + struct binding (`go test` included) |
| [environment](./environment/) | Environment-only config (`go test` included) |
| [webapp](./webapp/) | YAML defaults + `WEBAPP_*` overrides + binding + `Validate` + tests |
| [jsonschema](./jsonschema/) | `WithJSONSchema` on a file |
| [customvalidator](./customvalidator/) | `WithValidator` cross-field rule |
| [dump](./dump/) | `WithFileDumperAs` + `Dump` of merged state |
| [defaults](./defaults/) | `WithContent` defaults, then file, then env |
| [formats](./formats/) | `WithFileAs` with JSON + TOML |
| [consul](./consul/) | `WithConsulOptional` (no Consul required for tests) |
| [testing](./testing/) | `synthratest.Config` + `source.NewMap` in tests |

## Quick start

```bash
cd examples/basic && go run .
cd examples/environment && WEBAPP_SERVER_HOST=localhost WEBAPP_SERVER_PORT=8080 \
  WEBAPP_DATABASE_PRIMARY_HOST=db WEBAPP_DATABASE_PRIMARY_PORT=5432 \
  WEBAPP_DATABASE_PRIMARY_DATABASE=myapp WEBAPP_AUTH_JWT_SECRET=secret \
  WEBAPP_FEATURES_DEBUG_MODE=true go run .
cd examples/webapp && go run .
```

## Tests

Each example that ships tests passes with:

```bash
go test ./examples/...
```

## Docker (webapp)

From the repository root:

```bash
docker build -f examples/Dockerfile -t synthra-webapp-example .
docker run --rm synthra-webapp-example
```

The image uses the Go version pinned in [Dockerfile](./Dockerfile) (aligned with [go.mod](../go.mod)).

## Environment variable naming

Examples use explicit prefixes (`WEBAPP_`, `APP_`, `EDGE_`, `DEMO_`). Strip the prefix, split on `_`, lowercase, and nest: `WEBAPP_SERVER_READ_TIMEOUT` → `server.read.timeout`.

## Contributing a new example

1. Add a new subdirectory with `main.go`, `README.md`, and tests when behavior is non-trivial.
2. Link it from this file.
3. Keep snippets copy-paste accurate (`gopherly.dev/synthra` imports, real paths).
