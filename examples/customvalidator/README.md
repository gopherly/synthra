# Custom validator (`WithValidator`)

Uses `synthra.WithValidator` to enforce a rule on the **merged** configuration map after all sources load: when `server.tls.enabled` is true, both certificate and key paths must be non-empty.

```bash
cd examples/customvalidator && go run .
```

Fail the check deliberately:

```bash
cd examples/customvalidator && go run . config-invalid.yaml
```

Tests:

```bash
cd examples/customvalidator && go test -v
```
