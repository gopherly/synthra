# Environment variables example

Load all configuration from `WEBAPP_*` environment variables -- no files needed.

## What it shows

- `WithEnv("WEBAPP_")` as the only source
- Nested structs populated from underscore-separated variable names
- Direct key access with `cfg.String()`, `cfg.Int()`, `cfg.Bool()`
- Automatic string-to-type conversion

## Set the variables

```bash
export WEBAPP_SERVER_HOST=localhost
export WEBAPP_SERVER_PORT=8080
export WEBAPP_DATABASE_PRIMARY_HOST=db.example.com
export WEBAPP_DATABASE_PRIMARY_PORT=5432
export WEBAPP_DATABASE_PRIMARY_DATABASE=myapp
export WEBAPP_AUTH_JWT_SECRET=your-secret-key
export WEBAPP_FEATURES_DEBUG_MODE=true
```

## Run

```bash
cd examples/environment && go run .
```

## Tests

```bash
cd examples/environment && go test -v
```

## Expected output

```text
=== Simple Configuration ===
Server: localhost:8080
Database: db.example.com:5432/myapp
Auth JWT Secret: your-secret-key
Debug Mode: true
============================

=== Direct Configuration Access ===
Server: localhost:8080
Database: db.example.com
Debug mode is enabled
```

## How variable names map to keys

Strip the prefix, split on `_`, and lowercase each part.

| Environment variable           | Config path             | Struct field            |
|--------------------------------|-------------------------|-------------------------|
| `WEBAPP_SERVER_HOST`           | `server.host`           | `Server.Host`           |
| `WEBAPP_DATABASE_PRIMARY_HOST` | `database.primary.host` | `Database.Primary.Host` |
| `WEBAPP_AUTH_JWT_SECRET`       | `auth.jwt.secret`       | `Auth.JWT.Secret`       |

## Key ideas

1. **No files required** -- environment variables alone are enough.
2. **Underscore nesting** -- each `_` after the prefix creates a deeper level.
3. **Type conversion** -- string values are converted to the struct field's Go type.
4. **Direct access** -- you can also read keys with `cfg.String("server.host")` instead of binding.

## Docker example

This pattern follows the [Twelve-Factor App](https://12factor.net/config) methodology and works well with containers:

```bash
docker run -e WEBAPP_SERVER_HOST=0.0.0.0 \
           -e WEBAPP_SERVER_PORT=8080 \
           -e WEBAPP_DATABASE_PRIMARY_HOST=prod-db \
           -e WEBAPP_AUTH_JWT_SECRET=prod-secret \
           your-app
```
