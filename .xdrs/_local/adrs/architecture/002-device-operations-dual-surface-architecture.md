# _local-adr-002: Device Operations Dual-Surface Architecture

## Context and Problem Statement

The feature requires the same device operations to be available through both a REST API and an MCP server. The repository currently has a single-process Go server using `net/http` and an in-process bridge package.

Question: How should REST and MCP be added so both surfaces stay behaviorally consistent without duplicating business logic?

## Decision Outcome

**One shared in-process operations service with thin REST and MCP adapters mounted into the existing `net/http` server**

### Implementation Details

- Add a shared `operations` package that owns current-state reads, history queries, desired-state publication, and registration.
- Add an `auth` package that parses bearer tokens and evaluates `i:` and `r:` scopes.
- Add a thin `api` package for HTTP request validation and JSON responses.
- Add a thin `mcpapi` package using `github.com/mark3labs/mcp-go`, mounted on the same HTTP server at `/mcp`.
- Both transport adapters call the same `operations` methods so parity rules live in one place.
- The service remains in the existing `stutzthings-server` process beside the bridge runtime and `/health` endpoint.

## Considered Options

- (CHOSEN) **Shared service plus REST/MCP adapters**
  - Reason: Satisfies the spec's parity requirement and keeps logic centralized.
- (REJECTED) **Separate REST and MCP implementations**
  - Reason: High risk of behavioral drift and duplicate tests.
- (REJECTED) **Separate MCP sidecar process**
  - Reason: Adds deployment complexity without a current scaling or isolation need.

## References

- Feature spec: [specs/002-device-ops-api/spec.md](../../../../specs/002-device-ops-api/spec.md)
- Feature plan: [specs/002-device-ops-api/plan.md](../../../../specs/002-device-ops-api/plan.md)
- Related: [architecture/001-bridge-process-topology.md](001-bridge-process-topology.md)