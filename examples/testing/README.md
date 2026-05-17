# Testing helpers

Build configuration in tests without touching the filesystem. Use [`source.NewMap`](https://pkg.go.dev/gopherly.dev/synthra/source#NewMap) to provide values from a plain Go map, and [`synthratest.Config`](https://pkg.go.dev/gopherly.dev/synthra/synthratest#Config) to get a ready-to-use `*synthra.Config` that fails the test on construction errors.

## Run the tests

```bash
cd examples/testing && go test -v
```

`go run .` prints a short pointer to the tests -- the real content is in `main_test.go`.

## Example

```go
cfg := synthratest.Config(t,
    synthra.WithSource(source.NewMap(map[string]any{
        "server": map[string]any{"port": 8080, "host": "127.0.0.1"},
    })),
)
require.NoError(t, cfg.Load(t.Context()))
port, err := cfg.Int("server.port")
```

## Key ideas

1. **No files in tests** -- `source.NewMap` keeps tests fast, deterministic, and free from path issues.
2. **Test helper** -- `synthratest.Config` calls `t.Fatal` on error so your test stays clean.
3. **Same API** -- the config object you get back works exactly like a production one.
