# Testing helpers

Application tests can build configuration without touching the filesystem by using [`source.NewMap`](https://pkg.go.dev/gopherly.dev/synthra/source#NewMap) with [`synthratest.Config`](https://pkg.go.dev/gopherly.dev/synthra/synthratest#Config) (see package [`gopherly.dev/synthra/synthratest`](https://pkg.go.dev/gopherly.dev/synthra/synthratest)).

```bash
cd examples/testing && go test -v
```

`go run .` prints a short pointer to the tests above.
