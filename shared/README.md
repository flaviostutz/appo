# shared

Libraries shared across two or more of `appo/`, `stutzthings/`, or `devices/<virtual>/`.

Each shared library lives in its own sub-directory with its own `Makefile`. Libraries are versioned independently.

## Architecture overview

One sub-directory per shared library. Each library exposes a stable public API consumed by other application folders. No application-specific logic belongs here.

## How to build

```bash
make build
```

## How to run

Shared libraries are not run directly. Import them into an application module. See each sub-directory README for usage examples.
