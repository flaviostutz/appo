# Plan Checklist: MQTT to InfluxDB Bridge Daemon

**Purpose**: Author self-check — validate requirement completeness, clarity, consistency, and measurability across all dimensions before opening a PR
**Created**: 2026-03-14
**Feature**: [spec.md](../spec.md) | [plan.md](../plan.md) | [data-model.md](../data-model.md)

---

## Requirement Completeness

- [ ] CHK001 - Are functional requirements defined for all 3 user stories (sensor ingestion, connection resilience, multi-tenant isolation)? [Completeness, Spec §User Stories]
- [ ] CHK002 - Are startup and shutdown lifecycle requirements specified — not just steady-state ingestion? [Completeness, Spec §FR-014, Gap]
- [ ] CHK003 - Are graceful shutdown requirements defined — what happens to buffered messages when the process stops? [Gap]
- [ ] CHK004 - Are health-check or observability requirements defined so the bridge's operational status is externally visible? [Gap]
- [ ] CHK005 - Are requirements specified for what happens when a required env var is missing or invalid at startup? [Completeness, data-model.md §BridgeConfig validation]

## Requirement Clarity

- [ ] CHK006 - Is "silently discard and log a warning" (FR-008, FR-009) unambiguous — is the log level, format, and required fields (topic, raw payload) fully specified? [Clarity, Spec §FR-008, FR-009]
- [ ] CHK007 - Is the boolean-vs-numeric distinction requirement (FR-003, Spec §US1-AC3) objectively measurable — is there a specified mapping that prevents `true`/`false` from being stored as `1`/`0`? [Clarity, Spec §FR-003]
- [ ] CHK008 - Is the `ts` field type and precision unambiguous — is "Unix epoch milliseconds" the only accepted format, or are other representations (ISO 8601, seconds) also valid? [Clarity, Spec §FR-004, data-model.md]
- [ ] CHK009 - Is "automatic reconnect without requiring a process restart" (FR-006) quantified — is there a target reconnect time bound, or only SC-002's 10-second post-broker-restart resumption? [Clarity, Spec §FR-006, SC-002]
- [ ] CHK010 - Is "exponential backoff" in FR-007 defined with concrete parameters (initial interval, max interval, multiplier) or delegated entirely to the EDR? [Clarity, Spec §FR-007, EDR-001]

## Requirement Consistency

- [ ] CHK011 - Does the plan summary consistently reference MQTT 3.1.1 with `github.com/eclipse/paho.mqtt.golang`, matching the spec, research, and XDRs? [Consistency, plan.md §Summary]
- [ ] CHK012 - Are the data-model.md discard conditions consistent with spec FR-008/FR-009 — specifically, is "empty or null payload" a separate discard condition from "JSON missing `value` field"? [Consistency, data-model.md §Discard conditions, Spec §FR-009]
- [ ] CHK013 - Is the `MaxBufferSize ≥ BatchSize` validation rule in data-model.md consistent with FR-013 and FR-015 — could a batch size larger than the buffer size create an unresolvable eviction loop? [Consistency, data-model.md §BridgeConfig, Spec §FR-013, FR-015]
- [ ] CHK014 - Are the two "Language/Version" lines in plan.md a duplicate (copy-paste artifact)? [Consistency, plan.md §Technical Context]

## Acceptance Criteria Quality

- [ ] CHK015 - Can SC-001 ("appears in InfluxDB within 2 seconds") be objectively measured end-to-end — is the measurement point (MQTT publish vs. bridge receipt) specified? [Measurability, Spec §SC-001]
- [ ] CHK016 - Can SC-003 ("≥500 msg/s without measurable data loss") be objectively verified — is "measurable data loss" defined with a concrete threshold (e.g., 0 messages dropped, or ≤0.1%)? [Measurability, Spec §SC-003]
- [ ] CHK017 - Does User Story 2's independent test (stop broker 10s, restart, publish, verify) fully validate SC-002's "resumption within 10 seconds of broker becoming available"? [Acceptance Criteria, Spec §US2]

## Scenario Coverage

- [ ] CHK018 - Are requirements defined for the concurrent-message scenario — two messages arriving simultaneously for the same account/device/attribute with conflicting types? [Coverage, Spec §FR-003, Edge Cases]
- [ ] CHK019 - Are requirements defined for the `/set` topic direction received by the wildcard subscription — is the discard-with-warning behavior explicitly stated for 6-segment topics? [Coverage, Spec §Assumptions, Gap]
- [ ] CHK020 - Are recovery flow requirements defined after MaxWriteRetries is exhausted — does the bridge continue accepting new messages, or is there a circuit-breaker state? [Coverage, Spec §FR-007, Gap]

## Non-Functional Requirements

- [ ] CHK021 - Are memory usage bounds specified — with a 10,000-message ceiling and Go struct overhead, is there a stated maximum memory footprint? [Non-Functional, Gap]
- [ ] CHK022 - Is the security note about plain HTTP InfluxDB connections formally documented as a deployment prerequisite, not just an assumption comment? [Non-Functional, Spec §Assumptions, Security]
- [ ] CHK023 - Are logging requirements defined beyond "log a warning" — is there a required structured logging format, log fields schema, or log level policy? [Non-Functional, Gap]

## Dependencies & Assumptions

- [ ] CHK024 - Is the assumption "QoS 1 (at-least-once)" formally validated against the broker configuration — could the bridge receive duplicate messages, and if so, is idempotency required? [Assumption, Spec §Assumptions]
- [ ] CHK025 - Is the assumption "InfluxDB bucket/database pre-provisioned" documented as an explicit deployment prerequisite in the quickstart or README? [Dependency, quickstart.md, Spec §Assumptions]
- [ ] CHK026 - Is the dependency on `paho.mqtt.golang`'s `OnConnect` re-subscription pattern documented as a constraint — if the library changes its reconnect behavior, would the bridge silently stop re-subscribing? [Dependency, research.md §Decision 1]

## Ambiguities & Conflicts

- [ ] CHK027 - Is the detection order for scalar payloads (Boolean → Integer → Float → String) unambiguous for edge cases — e.g., is `"1"` a float or a string? Is `"true"` a boolean or a string? [Ambiguity, data-model.md §Payload Parsing Rules]
- [ ] CHK028 - Is the FIFO eviction unit clear — does "evict the oldest `BatchSize` messages" mean one full batch is evicted per overflow event, or exactly one message? [Ambiguity, Spec §FR-015, EDR-001]
