# appo

Agentic AI module (Python, LangGraph). The Appo agent itself — receives events from external sources (MQTT, webhooks, timers), maintains persistent memory, reads instruction files, and reacts through tool calls.

One sub-directory per agent or agent group.

## Architecture overview

This application folder contains one sub-directory per agent or agent group. Each agent sub-module is independently runnable, maintains its own memory, and interacts with the external world through tool calls via the MCP interface exposed by `stutzthings/`.

## How to build

```bash
make build
```

## How to run

```bash
make test
```

> Full instructions live inside each agent sub-directory once modules are added.
