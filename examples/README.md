# Synthra examples

Runnable programs that complement the shorter snippets in [example_test.go](../example_test.go) and the package overview on [pkg.go.dev](https://pkg.go.dev/gopherly.dev/synthra).

## Progression

Start at the top and work down. Each example builds on concepts from the previous ones.

| Directory | What it shows |
|-----------|---------------|
| [basic](./basic/) | YAML file + struct binding |
| [webapp](./webapp/) | YAML defaults + `WEBAPP_*` env overrides + binding + `Validate` |
| [testing](./testing/) | `synthratest.Config` + `source.NewMap` in tests |
| [schema](./schema/) | `WithJSONSchema` validation + schema `default` values + `patternProperties` |
| [casing](./casing/) | Case-insensitive merge, schema as canonical key-casing authority |
| [hooks](./hooks/) | `WithTransform`, `WithValidator`, and `OnBound[T]` in one pipeline |
| [codecs](./codecs/) | `WithFileAs` (JSON, TOML) + `WithFileDumperAs` (YAML dump) |
| [envsubst-layered](./envsubst-layered/) | Three-layer `Resolver.Or` precedence for `${VAR}` lookups |
| [multi-schema](./multi-schema/) | Two-phase `WithJSONSchemaFunc` around `WithEnvSubst` |
| [consul](./consul/) | `WithIf(..., WithConsul(...))` conditional source |

## Quick start

```bash
cd examples/basic && go run .
cd examples/webapp && go run .
```

## Tests

Every example ships with tests. Run them all at once:

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

## Adding a new example

1. Create a subdirectory with `main.go`, `README.md`, and tests.
2. Link it from this file.
3. Keep snippets copy-paste accurate (`gopherly.dev/synthra` imports, real paths).
