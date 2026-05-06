# Embedded defaults (`WithContent`)

Sources are merged in order; later values override earlier ones.

1. `WithContent` — small baked-in YAML defaults  
2. `WithFile("overrides.yaml")` — checked-in overrides  
3. `WithEnv("DEMO_")` — highest precedence (for example `DEMO_SERVER_PORT`)

```bash
cd examples/defaults
DEMO_SERVER_PORT=9999 go run .
```

Tests:

```bash
cd examples/defaults && go test -v
```
