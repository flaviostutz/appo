<!--
Sync Impact Report
==================
Version change: 1.0.0 → 1.1.0 (MINOR — new Principle VI added)

Modified principles: N/A

Added sections:
  - Principle VI: XDR-First Feature Documentation

Removed sections: N/A

Templates requiring updates:
  - .specify/templates/plan-template.md  ✅ updated — XDR Sync section added
  - .specify/templates/spec-template.md  ✅ updated — XDR Candidates section added
  - .specify/templates/tasks-template.md ✅ checked — no static references requiring update
  - .specify/templates/agent-file-template.md ✅ checked — generic placeholders only; no updates required
  - .specify/templates/checklist-template.md  ✅ checked — generic placeholders only; no updates required
  - .github/agents/speckit.specify.agent.md ✅ updated — XDR candidate identification step added
  - .github/agents/speckit.plan.agent.md ✅ updated — XDR creation/update step added to Phase 1
  - .xdrs/_local/adrs/index.md ✅ created — stub for local ADR index
  - .xdrs/_local/edrs/index.md ✅ created — stub for local EDR index

Follow-up TODOs: None — all fields resolved.
-->

# appo Constitution

## Core Principles

### I. XDR-Driven Decision Making

All implementation decisions MUST be preceded by consulting applicable XDRs listed in `.xdrs/index.md`.
Coding agents and human contributors MUST follow the XDR hierarchy defined there, with `_local` scope
overriding all other scopes where conflicts exist.
Any deviation from an XDR MUST be documented as a `_local` XDR before code is written — not after.
XDRs are the single source of truth; ad-hoc undocumented conventions are forbidden.

*Rationale*: Prevents undocumented decisions, reduces drift between codebases, and ensures coding agents
and human contributors operate from the same authoritative source.

### II. Monorepo Structure

The repository MUST follow the standard monorepo layout defined in agentkit-edr-005:
top-level application folders (`appo/`, `stutzthings/`, `devices/`, `deployments/`, `shared/`),
a root-level `shared/` area for resources consumed by more than one application,
Mise-managed tool versions via `.mise.toml` at the repository root,
and a `Makefile` at the repository root and inside every application and module folder.

Each Makefile MUST expose at minimum: `all`, `build`, `lint`, and `test` targets (agentkit-edr-008).
Applications MUST NOT depend directly on other applications; cross-application sharing MUST go
through `shared/` or published artifacts (container images, published libraries).
All folder and file names MUST be lowercase with hyphens as word separators.

*Rationale*: A predictable layout lowers onboarding cost and makes cross-module CI uniform and automatable.

### III. Quality Standards

Every module MUST meet the minimum baseline defined in agentkit-edr-007:

- `README.md` MUST include a Getting Started section with a runnable example in the first 20 lines.
- Unit tests MUST run automatically before every release (`make test`) and MUST block release on failure.
- Test coverage MUST reach at least 80% line and branch coverage, enforced in CI (agentkit-edr-004).
- `make lint` MUST run with zero-warning tolerance; lint failures MUST block CI builds.
- The project MUST comply with all applicable workspace XDRs; deviations require a `_local` XDR.

*Rationale*: A non-negotiable quality baseline prevents regressions and makes automated quality gates reliable.

### IV. Coding Best Practices

Source files MUST NOT exceed 400 lines; files that grow beyond this limit MUST be split by responsibility
(agentkit-edr-002). When a logical section inside a function exceeds approximately 20 lines, it MUST be
extracted into a named helper function (Template Method pattern).
README files, tests, and usage examples MUST be kept in sync with every implementation change.
Error handling MUST follow agentkit-edr-009: catch only where genuine recovery is possible, expose errors
as return values at module boundaries, and never swallow exceptions silently — intentional suppression
MUST be documented with an inline comment explaining why.

*Rationale*: Small files and named helpers reduce cognitive load, speed up review, and make failures diagnosable.

### V. CI/CD & Versioning

Projects MUST use three separate GitHub Actions workflows (agentkit-edr-006):
`ci.yml` triggered on PRs and pushes to `main` (runs `build`, `lint`, `test`);
`release.yml` triggered manually (`workflow_dispatch`) to calculate and push a semver tag via monotag;
`publish.yml` triggered on tag push to publish build artifacts.
All releases MUST be tagged `<module-name>/<semver>` (e.g., `appo/1.0.0`, `stutzthings/1.2.3`).

*Rationale*: Decoupled workflows prevent accidental publishes, make each lifecycle phase independently
auditable, and enforce consistent semver discipline across the monorepo.

### VI. XDR-First Feature Documentation

Every feature going through the speckit workflow MUST produce XDRs that capture durable decisions
extracted from the specification and design artifacts. XDRs are the authoritative source of truth;
`specs/` documents are transitional working artefacts and MUST NOT be the primary reference once
XDRs are written.

**What to capture in each type:**
- **BDRs** (`_local/bdrs/`): Product and business decisions — data models, tenant isolation rules,
  topic/payload schemas, access control policies, feature behaviour contracts.
- **ADRs** (`_local/adrs/`): Architectural decisions — component boundaries, integration patterns,
  protocol choices, inter-service communication, deployment topology.
- **EDRs** (`_local/edrs/`): Engineering conventions — implementation patterns, error handling
  strategies, retry/backoff policies, testing discipline, configuration management.

**Workflow obligations:**
- During `speckit.specify`: identify candidate XDR topics from functional requirements and key
  entities; list them in a "XDR Candidates" section of the spec.
- During `speckit.plan`: create or update BDRs, ADRs, and EDRs in `.xdrs/_local/` from
  design decisions made in Phase 0 (research) and Phase 1 (design/contracts).
- XDR documents MUST be short: one decision per file, ≤ 2 screens. Use bullet lists over prose.
- If an applicable XDR already exists, update it rather than creating a duplicate.
- Update the corresponding `_local/{type}/index.md` whenever an XDR is created or renamed.

*Rationale*: Feature specs are point-in-time working documents; XDRs are durable, searchable, and
discoverable by all contributors and coding agents. Centralising decisions in `.xdrs/_local/`
eliminates knowledge fragmentation across spec files, READMEs, and pull requests.

## Technology Stack & Tools

Approved technology choices for this project:

| Module | Language / Runtime | Key Frameworks & Tools |
|--------|--------------------|------------------------|
| `appo/` | Python | LangGraph, MCP client |
| `stutzthings/stutzthings-server` | Go | REST API, MQTT bridge, InfluxDB |
| `stutzthings/stutzthings-sdk-python` | Python | MCP client, MQTT SDK |
| `devices/esp32-*` | C/C++ (PlatformIO) | ESP-IDF, FreeRTOS |
| `devices/alexa` | Python | Alexa integration library |
| `deployments/` | Docker Compose, YAML | Raspberry Pi + cloud |
| Tooling (all modules) | — | Mise, Make, GitHub Actions, monotag |

New technology additions MUST be justified via an XDR in the `_local` scope before use in the codebase.
Ad-hoc technology introductions without an XDR are forbidden.

## Governance

This constitution supersedes all other undocumented conventions and informal practices.
It is the starting point for all plan Constitution Checks (see `.specify/templates/plan-template.md`).

**Amendment versioning** follows semantic versioning rules:
- MAJOR: removal or backward-incompatible redefinition of an existing principle.
- MINOR: new principle, section, or materially expanded guidance.
- PATCH: clarifications, wording fixes, or non-semantic refinements.

All PRs and agent-generated implementations MUST be verified against this constitution before merge.
Compliance is validated during code review and by the `001-lint` skill.
Changes to this constitution MUST update `.specify/memory/constitution.md` and be propagated to any
dependent template sections that contain static references to constitution principles.

**Version**: 1.1.0 | **Ratified**: 2026-03-14 | **Last Amended**: 2026-03-14
