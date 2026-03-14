---
description: Execute the implementation planning workflow using the plan template to generate design artifacts.
handoffs: 
  - label: Create Tasks
    agent: speckit.tasks
    prompt: Break the plan into tasks
    send: true
  - label: Create Checklist
    agent: speckit.checklist
    prompt: Create a checklist for the following domain...
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

1. **Setup**: Run `.specify/scripts/bash/setup-plan.sh --json` from repo root and parse JSON for FEATURE_SPEC, IMPL_PLAN, SPECS_DIR, BRANCH. For single quotes in args like "I'm Groot", use escape syntax: e.g 'I'\''m Groot' (or double-quote if possible: "I'm Groot").

2. **Load context**: Read FEATURE_SPEC and `.specify/memory/constitution.md`. Load IMPL_PLAN template (already copied).

3. **Execute plan workflow**: Follow the structure in IMPL_PLAN template to:
   - Fill Technical Context (mark unknowns as "NEEDS CLARIFICATION")
   - Fill Constitution Check section from constitution
   - Evaluate gates (ERROR if violations unjustified)
   - Phase 0: Generate research.md (resolve all NEEDS CLARIFICATION)
   - Phase 1: Generate data-model.md, contracts/, quickstart.md
   - Phase 1: Update agent context by running the agent script
   - Re-evaluate Constitution Check post-design

4. **Stop and report**: Command ends after Phase 2 planning. Report branch, IMPL_PLAN path, and generated artifacts.

## Phases

### Phase 0: Outline & Research

1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:

   ```text
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

### Phase 1: Design & Contracts

**Prerequisites:** `research.md` complete

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Define interface contracts** (if project has external interfaces) → `/contracts/`:
   - Identify what interfaces the project exposes to users or other systems
   - Document the contract format appropriate for the project type
   - Examples: public APIs for libraries, command schemas for CLI tools, endpoints for web services, grammars for parsers, UI contracts for applications
   - Skip if project is purely internal (build scripts, one-off tools, etc.)

3. **Agent context update**:
   - Run `.specify/scripts/bash/update-agent-context.sh copilot`
   - These scripts detect which AI agent is in use
   - Update the appropriate agent-specific context file
   - Add only new technology from current plan
   - Preserve manual additions between markers

4. **XDR sync** (Constitution Principle VI — mandatory):
   - Read the "XDR Candidates" section from the feature spec (added by `speckit.specify`).
   - For each candidate, check `.xdrs/_local/` for an existing XDR covering the same decision.
   - **If an XDR already exists**: update it in-place with any new details from this feature's
     design; keep the document short (≤ 2 screens, one decision per file).
   - **If no XDR exists**: create a new file in the appropriate subdirectory:
     - BDR → `.xdrs/_local/bdrs/<subject>/<NNN>-<slug>.md`
     - ADR → `.xdrs/_local/adrs/<subject>/<NNN>-<slug>.md`
     - EDR → `.xdrs/_local/edrs/<subject>/<NNN>-<slug>.md`
   - Use the existing `.xdrs/_local/bdrs/product/001-mqtt-topic-structure.md` as the style
     reference (Context, Decision Outcome, Implementation Details, Considered Options, References).
   - After creating/updating each XDR, add or update its entry in the matching index
     (`.xdrs/_local/{bdrs|adrs|edrs}/index.md`).
   - Add a reference back to the feature spec and plan from the XDR's References section.

**Output**: data-model.md, /contracts/*, quickstart.md, agent-specific file, updated `.xdrs/_local/`

## Key rules

- Use absolute paths
- ERROR on gate failures or unresolved clarifications
