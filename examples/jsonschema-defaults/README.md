# JSON Schema defaults and interpolation

Synthra automatically applies `"default"` values declared in a JSON Schema to any missing keys after loading sources. This includes defaults nested inside `"properties"` and `"patternProperties"`. Combined with `WithInterpolation`, you can substitute `{key}` placeholders in string values before validation.

## What it shows

- `WithJSONSchema(schema)` filling in missing top-level and nested keys from `"default"` declarations.
- `patternProperties` defaults applied to every existing matching key (the components map).
- `WithInterpolation` substituting `{env}` placeholders in string values.
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

## Run

```bash
cd examples/jsonschema-defaults && go run .
```

## Tests

```bash
cd examples/jsonschema-defaults && go test -v
```
