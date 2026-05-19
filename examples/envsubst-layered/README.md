# Envsubst layered resolver example

Demonstrates three-layer variable precedence using `Resolver.Or` for `${VAR}` placeholder expansion.

## What it shows

- `synthra.FromEnv().Prefix("DPY_VAR_")`: highest priority, operator-level OS environment overrides
- `synthra.FromEnvFile(path)`: middle priority, per-environment `.env` file
- `synthra.FromMap(defaults)`: lowest priority, static fallback values embedded in the manifest

## Variable resolution order

```text
DPY_VAR_* OS env  (highest)
  ↓ if not found
.env file          (middle)
  ↓ if not found
manifest defaults  (lowest)
```

## The config

`config.yaml` uses `${VAR}` and `${VAR:-fallback}` placeholders:

```yaml
service: my-app
region: ${REGION}
image: my-app:${TAG:-latest}
db_url: postgres://${DB_HOST:-localhost}:5432/mydb
```

## Run

```bash
cd examples/envsubst-layered && go run .
```

Override a variable at operator level:

```bash
DPY_VAR_REGION=us-east-1 go run .
```

## Tests

```bash
cd examples/envsubst-layered && go test -v
```

## Key ideas

1. `Resolver.Or` chains resolvers so the first one that finds the variable wins.
2. This pattern separates three concerns: operator overrides, per-environment config, and code defaults.
3. The `.env` file is optional. When absent, resolution falls through to static defaults without error.
