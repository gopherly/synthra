# Hooks example

Demonstrates all three hook points in one pipeline, showing the exact order they run.

## What it shows

- `WithTransform`: map stage, inspect and mutate the raw key-value map before binding
- `WithValidator`: cross-field, validate relationships between fields (TLS cert + key consistency)
- `OnBound[T]`: struct stage, post-binding normalization (lowercase the log level)

## Pipeline order

```text
Load sources -> WithTransform -> WithValidator -> WithBinding -> OnBound[T]
```

## Run

```bash
cd examples/hooks && go run .
```

## Tests

```bash
cd examples/hooks && go test -v
```

The tests cover:

- Happy path with all hooks running
- `WithTransform` overriding log level for production
- `WithValidator` rejecting TLS enabled without cert/key files
- `OnBound` normalizing the log level to lowercase

## Key ideas

1. Hook points are **ordered**: transforms run before validators, which run before binding, which runs before `OnBound`.
2. `OnBound[T]` is **type-safe**: passing a mismatched `OnBound[Other]` is a compile error, not a runtime panic.
3. `WithValidator` operates on the raw `*Values` map, making cross-field checks easy without Go struct reflection.
