# Optional Consul layer

This program always loads `config.yaml`, then **optionally** loads a YAML value from Consul when `CONSUL_HTTP_ADDR` is set, then applies `EDGE_*` environment variables (highest precedence).

Without Consul, wrapping Consul with `WithIf` makes it a no-op:

```bash
cd examples/consul && go run .
```

With Consul (illustrative — adjust the KV path to your cluster):

```bash
export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
# write YAML to KV at synthra/example/config.yaml (or change the path in main.go)
cd examples/consul && go run .
```

Example `docker compose` for a local Consul agent is left to your infrastructure; the client only needs `CONSUL_HTTP_ADDR` as documented in [synthra.WithConsul](https://pkg.go.dev/gopherly.dev/synthra#WithConsul), used conditionally with `WithIf`.

Tests (no live Consul required):

```bash
cd examples/consul && go test -v
```
