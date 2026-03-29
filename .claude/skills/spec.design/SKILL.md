---
name: spec.design
description: Create a technical design document for an existing spec. Reads the Requirements note and generates architecture, data model, and interface decisions. Run after /spec and before /spec.tasks.
argument-hint: "[SPEC-NNN]"
disable-model-invocation: true
user-invocable: true
effort: high
allowed-tools: Bash(ctxpin *), Read, Glob, Grep, Edit, Write, AskUserQuestion
---

# Design — Create Technical Design

You are a technical architect. Your job is to read the Requirements note from an existing spec and create a **Design** note with architecture decisions, data model, and interface contracts.

## Input

Arguments: **$ARGUMENTS**

## Step 1: Resolve Spec

**If `SPEC-NNN` is provided:** Find the spec folder whose `_frontmatter.yml` has that label.

**If no arguments:** Infer from the active ContextPin context. Look for a `SPEC-*` label in any note currently in context (injected by the ADE hook). If no context is available, list specs and ask:

```bash
ctxpin notes list Specs --json
```

### Locate the Spec Folder

List folders in `Specs/`, read each `_frontmatter.yml` to find the one matching the `SPEC-NNN` label.

## Step 2: Validate Prerequisites

1. Read the Requirements note from the spec folder
2. If the Requirements note doesn't exist or is empty, stop and tell the user to run `/spec` first
3. Check if a Design note already exists — if so, warn the user and ask if they want to overwrite or update it

## Step 3: Read Context

1. **Requirements note** — understand what needs to be built
2. **Codebase** — read existing code to understand current architecture, patterns, conventions, and what already exists
3. **Planning docs** — check `docs/final-plan/` for relevant architecture and design documents
4. **Other specs** — check if other specs have Design notes with patterns to stay consistent with

## Step 4: Create Design Note

If the Design note doesn't exist yet, create it:

```bash
ctxpin notes create note "Design" --folder "Specs/<Feature Name>" --json
```

Add the `SPEC-NNN` label to the note's frontmatter.

## Step 5: Write Design

Edit the **Design** note with this structure:

```markdown
# Design

## Approach
[High-level technical approach — 2-3 sentences on the chosen strategy and WHY this approach over alternatives]

## Architecture
[Key architectural decisions and rationale. Reference existing codebase patterns.
- Where does this feature fit in the current architecture?
- What existing components does it extend or interact with?
- What new components are needed?]

## Data Model
[Entities, relationships, and key fields.
- New types/structs with field definitions
- Relationships to existing models
- Migration or schema changes needed]

## API / Interfaces
[Key interfaces, endpoints, or contracts.
- Function/method signatures
- HTTP endpoints if applicable
- Internal interfaces between components
- Error types and handling contracts]

## Dependencies
[External libraries, services, or internal modules this feature depends on.
- Existing internal packages to reuse
- New external dependencies needed (with justification)
- Integration points with other systems]

## Risks & Mitigations
[Known risks and how to address them.
- Technical debt implications
- Performance concerns
- Security considerations
- Backward compatibility]
```

**Writing guidelines:**
- Reference specific files and packages from the codebase (`internal/pkg/...`, `cmd/...`)
- Explain WHY each decision was made, not just what
- Keep it actionable — a developer should be able to start implementing from this
- Stay consistent with patterns in other spec Design notes
- Don't repeat requirements — reference them by ID (e.g., "To satisfy FR-001...")

## Step 6: Report

After creating the design, report:
- The spec it belongs to (SPEC-NNN, feature name)
- Key architectural decisions made
- New components or packages proposed
- External dependencies needed
- Remind the user: **review the design, then run `/tasks` to generate the task breakdown**

## Important Rules

1. **Always use `ctxpin` CLI** to create notes — never create files manually
2. **Always add the `SPEC-NNN` label** to the Design note
3. **Read Requirements first** — design must trace back to requirements
4. **Read the codebase** — design must build on existing patterns, not invent new ones
5. **Only create the Design note** — do NOT create or modify Requirements or Tasks
6. **Be specific** — name real files, packages, and interfaces from the codebase
