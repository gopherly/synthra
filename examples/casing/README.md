# Casing example

Demonstrates Synthra's case-insensitive, case-preserving merge and shows how a JSON Schema acts as the canonical authority for key casing.

## What it shows

- Without a schema, the first source's key casing is preserved and all other sources merge values into it case-insensitively.
- With a JSON Schema, Synthra renames keys to match the schema's `"properties"` declarations before validation runs. The schema is the casing authority.

## The configs

`config-base.yaml` uses mixed casing (`ApiVersion: v1`). `config-override.yaml` uses canonical camelCase (`apiVersion: v2`).

Without a schema:

```text
ApiVersion: v2   # first-source casing preserved, value overridden by second source
```

With a schema declaring `"apiVersion"`:

```text
apiVersion: v2   # key renamed to schema casing
```

## Run

```bash
cd examples/casing && go run .
```

## Tests

```bash
cd examples/casing && go test -v
```

## Key ideas

1. Case-insensitive merge means `ApiVersion`, `apiversion`, and `APIVERSION` all resolve to the same value.
2. Without a schema, whichever source loads first sets the canonical key casing.
3. A JSON Schema with `"properties"` makes the schema the single source of truth for key names. This is useful when config files come from different teams with inconsistent conventions.
