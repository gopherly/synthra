# Web application example (layered config)

This example shows a realistic pattern: **YAML defaults**, **`WEBAPP_*` environment overrides**, **struct binding** with explicit `synthra` tags, and a **`Validate` method** on the bound struct ([synthra.Validator](https://pkg.go.dev/gopherly.dev/synthra#Validator)).

## Features

- Mixed sources: `WithFile` then `WithEnv` (later source wins on conflicts)
- Nested YAML matching nested struct tags (`server.read.timeout`, and so on)
- Direct key access with dot paths (`cfg.String("server.host")`)
- Tests for env-only, YAML-only, layered precedence, and validation failures

## Run

```bash
cd examples/webapp
go run .
```

## Tests

```bash
cd examples/webapp
go test -v
```

## Optional: load the same keys from your shell

```bash
source examples/webapp/setup_env.sh
cd examples/webapp && go run .
```

## Environment variables

Strip the `WEBAPP_` prefix, split the remainder on `_`, and nest keys in lowercase. Examples:

| Variable | Config key |
|----------|------------|
| `WEBAPP_SERVER_PORT` | `server.port` |
| `WEBAPP_SERVER_READ_TIMEOUT` | `server.read.timeout` |
| `WEBAPP_DATABASE_PRIMARY_HOST` | `database.primary.host` |
| `WEBAPP_FEATURES_DEBUG_MODE` | `features.debug.mode` |

## Production-style construction

```go
cfg, err := synthra.New(
    synthra.WithFile("config.yaml"),
    synthra.WithEnv("WEBAPP_"),
    synthra.WithBinding(&appConfig),
)
if err != nil {
    log.Fatal(err)
}
if err := cfg.Load(ctx); err != nil {
    log.Fatal(err)
}
```

## Docker (optional)

From the **repository root** (see [examples/Dockerfile](../Dockerfile)):

```bash
docker build -f examples/Dockerfile -t synthra-webapp-example .
docker run --rm synthra-webapp-example
```

Override ports or hosts at runtime with `-e WEBAPP_SERVER_PORT=8080`, and so on.
