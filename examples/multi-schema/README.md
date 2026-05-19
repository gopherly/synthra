# Multi-schema example

Demonstrates two-phase JSON Schema validation using `WithJSONSchemaFunc` around `WithEnvSubst`.

## What it shows

- Validate the raw manifest against a partial schema **before** variable substitution.
- Expand `${VAR}` placeholders with `WithEnvSubst`.
- Validate the fully-substituted manifest against the complete schema **after** substitution.

## Why two schemas?

Some fields must be structurally valid before substitution (e.g. the `environments` array must exist as a concrete list). Other fields may contain placeholders before substitution (e.g. `service: ${APP_SERVICE}`). Running one schema before and one after substitution enforces both constraints cleanly.

## Pipeline

```text
Load manifest.yaml
  → WithJSONSchemaFunc (validates environments block, no placeholders allowed)
  → WithEnvSubst (expands ${VAR} from OS environment)
  → WithJSONSchemaFunc (validates complete substituted manifest)
```

## Run

```bash
cd examples/multi-schema && go run .
```

With a custom service name:

```bash
APP_SERVICE=my-api go run .
```

## Tests

```bash
cd examples/multi-schema && go test -v
```

## Key ideas

1. `WithJSONSchemaFunc` takes a function `func(*Values) ([]byte, error)` so the schema can be chosen dynamically based on already-loaded values.
2. Placing a schema step **before** `WithEnvSubst` catches unexpanded placeholders in required fields.
3. Placing a schema step **after** `WithEnvSubst` validates the final resolved values.
4. `synthra.ConfigError` carries a `Path` field indicating which step failed. Useful for diagnostic messages.
