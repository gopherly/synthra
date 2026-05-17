# JSON Schema defaults and variable substitution

Synthra automatically applies `"default"` values declared in a JSON Schema to any missing keys after loading sources. This includes defaults nested inside `"properties"` and `"patternProperties"`. Combined with `WithEnvSubst`, you can expand `${VAR}` placeholders in string values before validation runs.

## What it shows

- `WithJSONSchema(schema)` filling in missing top-level and nested keys from `"default"` declarations.
- `patternProperties` defaults applied to every existing matching key (the components map).
- `WithEnvSubst` expanding `${ENV}` placeholders in string values after schema defaults are applied.
- User-provided values are never overridden by schema defaults.

## The config

`config.yaml` is intentionally minimal — only required and non-default values are specified:

```yaml
service: my-app
server:
  max_connections: 50
components:
  web:
    image: nginx:1.27
  worker:
    image: my-app:latest
    role: worker
    replicas: 3
```

## The schema (key excerpts)

```json
{
  "required": ["service"],
  "properties": {
    "port":      { "default": 8080 },
    "log_level": { "default": "info" },
    "server": {
      "properties": {
        "timeout":         { "default": "30s" },
        "max_connections": { "default": 100 }
      }
    },
    "components": {
      "patternProperties": {
        "^[a-z0-9-]+$": {
          "properties": {
            "role":     { "default": "service" },
            "replicas": { "default": 1 }
          }
        }
      }
    }
  }
}
```

## After loading

| Key | Source |
|-----|--------|
| `service` | config.yaml |
| `port` | schema default (8080) |
| `log_level` | schema default (info) |
| `server.timeout` | schema default (30s) |
| `server.max_connections` | config.yaml (50) |
| `components.web.role` | schema patternProperties default (service) |
| `components.web.replicas` | schema patternProperties default (1) |
| `components.worker.role` | config.yaml (worker) |
| `components.worker.replicas` | config.yaml (3) |

## Variable substitution with WithEnvSubst

The program sets `ENV=production` and passes it through `resolve.Vars`. If a config field contains `${ENV}`, it is expanded to `production` after schema defaults are applied.

For example, if you add `envfile: ".env.${ENV}"` to `config.yaml`, it becomes `.env.production` after `Load`.

Use `${VAR:-fallback}` to provide a default for variables that may not be set:

```go
synthra.WithEnvSubst(
    resolve.Vars(map[string]string{"ENV": env}),
    resolve.OSPrefix("APP_"),  // higher priority overrides
)
```

See the [resolve package](../../resolve) for all available resolver constructors.

## Run

```bash
cd examples/jsonschema-defaults && go run .
```

## Tests

```bash
cd examples/jsonschema-defaults && go test -v
```
