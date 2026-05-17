# JSON Schema validation

Validate the merged configuration against a JSON Schema before your program uses it. If a value has the wrong type or a required key is missing, `Load` returns an error right away.

## What it shows

- `WithJSONSchema(schema)` applied to a file source
- A valid config that passes the schema
- An invalid config (`config-invalid.yaml`) that is rejected at load time

## The schema

`schema.json` requires three keys: `service` (non-empty string), `port` (integer 1--65535), and `log_level` (one of `debug`, `info`, `warn`, `error`).

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["service", "port", "log_level"],
  "properties": {
    "service": { "type": "string", "minLength": 1 },
    "port": { "type": "integer", "minimum": 1, "maximum": 65535 },
    "log_level": { "type": "string", "enum": ["debug", "info", "warn", "error"] }
  },
  "additionalProperties": true
}
```

## Run

```bash
cd examples/jsonschema && go run .
```

Output: `service=api port=8080 log_level=info`

## Try the invalid config

`config-invalid.yaml` sets `port` to the string `"not-a-number"`, which violates the schema. You can test this in the test suite -- the load fails with a schema error.

## Tests

```bash
cd examples/jsonschema && go test -v
```

The tests cover both the happy path and the rejection path.

## Key ideas

1. **Fail early** -- schema errors surface during `Load`, not later in your application.
2. **Works with any source** -- the schema validates the merged map, so it works with files, env vars, or both.
3. **Standard format** -- the schema follows [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/schema), so you can reuse it in editors, CI, or other tools.
