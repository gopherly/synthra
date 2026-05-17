# Optional Consul layer

Load `config.yaml` first, optionally pull a YAML value from Consul KV, then apply `EDGE_*` environment variables (highest precedence).

When `CONSUL_HTTP_ADDR` is not set, `WithIf` turns the Consul source into a no-op -- the program still runs using file and environment only.

## Run (without Consul)

```bash
cd examples/consul && go run .
```

## Run (with Consul)

Point `CONSUL_HTTP_ADDR` to a running Consul agent and make sure the KV path exists:

```bash
export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
cd examples/consul && go run .
```

Adjust the KV path in `main.go` (`synthra/example/config.yaml`) to match your cluster.

## Tests

No live Consul is required -- the tests only exercise the file + env path:

```bash
cd examples/consul && go test -v
```

## Key ideas

1. **Conditional sources** -- `WithIf(condition, option)` adds a source only when the condition is true.
2. **Graceful degradation** -- the program works with or without Consul.
3. **Same merge order** -- Consul sits between file and env, so environment variables still win.
