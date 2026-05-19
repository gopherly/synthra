# Web application example (layered config)

A realistic setup: **YAML defaults**, **`WEBAPP_*` environment overrides**, **struct binding**, and a **`Validate` method** on the bound struct.

## What it shows

- Layered sources -- `WithFile` first, then `WithEnv` (later source wins on conflicts)
- Deeply nested YAML matching nested struct tags (`server.tls.cert.file`, etc.)
- Direct key access with dot paths (`cfg.String("server.host")`)
- Struct-level validation via the [`synthra.Validator`](https://pkg.go.dev/gopherly.dev/synthra#Validator) interface
- Tests for env-only, YAML-only, layered precedence, and validation failures

## Run

```bash
cd examples/webapp && go run .
```

## Tests

```bash
cd examples/webapp && go test -v
```

## Load keys from your shell (optional)

```bash
source examples/webapp/setup_env.sh
cd examples/webapp && go run .
```

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

## Baking defaults into the binary

To embed defaults directly in the binary instead of shipping a separate `config.yaml`, prepend `synthra.WithContent` as the lowest-priority source:

```go
cfg := synthra.MustNew(
    synthra.WithContent(defaultYAML, codec.YAML), // lowest priority
    synthra.WithFile("config.yaml"),               // overrides defaults
    synthra.WithEnv("WEBAPP_"),                    // highest priority
    synthra.WithBinding(&appConfig),
)
```

The same merge rules apply: each source overrides keys from all earlier sources.

## How environment variables map to keys

Strip the `WEBAPP_` prefix, split on `_`, lowercase.

| Variable | Config key |
|----------|------------|
| `WEBAPP_SERVER_PORT` | `server.port` |
| `WEBAPP_SERVER_TLS_ENABLED` | `server.tls.enabled` |
| `WEBAPP_DATABASE_PRIMARY_HOST` | `database.primary.host` |
| `WEBAPP_AUTH_JWT_SECRET` | `auth.jwt.secret` |
| `WEBAPP_FEATURES_DEBUG_MODE` | `features.debug.mode` |

## Docker (optional)

From the **repository root** (see [examples/Dockerfile](../Dockerfile)):

```bash
docker build -f examples/Dockerfile -t synthra-webapp-example .
docker run --rm synthra-webapp-example
```

Override values at runtime with `-e WEBAPP_SERVER_PORT=8080`, and so on.
