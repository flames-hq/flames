---
name: spec.tasks
description: Generate an ordered task breakdown from a spec's Requirements and Design notes. Creates a checklist with TASK-NNN IDs, phases, dependencies, and parallelization hints. Run after /spec.design.
argument-hint: "[SPEC-NNN]"
disable-model-invocation: true
user-invocable: true
effort: high
allowed-tools: Bash(ctxpin *), Read, Glob, Grep, Edit, Write, AskUserQuestion
---

# Tasks — Generate Task Breakdown

You are a technical project planner. Your job is to read the Requirements and Design notes from an existing spec and create a **Tasks** note with an ordered, dependency-aware checklist of implementation tasks.

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

1. Read the **Requirements** note — if missing or empty, stop and tell user to run `/spec` first
2. Read the **Design** note — if missing or empty, stop and tell user to run `/design` first
3. Check if a Tasks note already exists — if so, warn the user and ask if they want to overwrite or update it

## Step 3: Read Context

1. **Requirements note** — extract user stories, functional requirements, acceptance criteria
2. **Design note** — extract architecture, data model, interfaces, dependencies
3. **Codebase** — understand what already exists, what files need to be created vs modified
4. **Other specs' Tasks notes** — check for consistency in task granularity and style

## Step 4: Create Tasks Note

If the Tasks note doesn't exist yet, create it:

```bash
ctxpin notes create note "Tasks" --folder "Specs/<Feature Name>" --json
```

Add the `SPEC-NNN` label to the note's frontmatter.

## Step 5: Generate Task Breakdown

Break the work into atomic tasks organized by phase. Think deeply about:
- What needs to happen first (setup, scaffolding)
- What is the core logic (models, services, business rules)
- What wires things together (API endpoints, CLI commands, event handlers)
- What polishes the feature (error handling, edge cases, tests, docs)

### Task Format

Each task follows this format:
```
- [ ] TASK-NNN [markers] — Description (specific file paths)
```

**Markers (optional, combine as needed):**
- `[P]` — can run in parallel with other `[P]` tasks in the same phase
- `[depends: TASK-NNN, TASK-NNN]` — must wait for these tasks to complete
- `[US-N]` — traces back to a specific user story from Requirements

### Phase Structure

Write the **Tasks** note with this structure:

```markdown
# Tasks

## Phase 1: Setup
- [ ] TASK-001 — [Scaffold new package/module structure] (`path/to/new/package/`)
- [ ] TASK-002 [P] — [Create configuration/constants] (`path/to/config.go`)

## Phase 2: Core Implementation
- [ ] TASK-003 [depends: TASK-001] [US-1] — [Implement core entity/model] (`path/to/model.go`)
- [ ] TASK-004 [depends: TASK-001] [P] [US-1] — [Implement core service/logic] (`path/to/service.go`)
- [ ] TASK-005 [depends: TASK-003] [US-2] — [Implement secondary feature] (`path/to/feature.go`)

## Phase 3: Integration
- [ ] TASK-006 [depends: TASK-004, TASK-005] — [Wire into API/CLI layer] (`path/to/handler.go`)
- [ ] TASK-007 [depends: TASK-006] [P] — [Add integration tests] (`path/to/integration_test.go`)

## Phase 4: Polish
- [ ] TASK-008 [depends: TASK-006] — [Error handling and edge cases]
- [ ] TASK-009 [P] — [Add documentation and examples]
```

### Task Writing Rules

1. **Atomic** — each task is one clear deliverable, completable in a single coding session
2. **Specific file paths** — include the exact files to create or modify in parentheses
3. **Dependency-aware** — add `[depends: ...]` when a task requires another to complete first
4. **Parallelizable** — mark `[P]` for tasks that can run concurrently (different files, no shared state)
5. **Traceable** — add `[US-N]` to link tasks back to user stories from Requirements
6. **TDD-friendly** — where applicable, test tasks should appear before or alongside implementation tasks
7. **Right-sized** — aim for 4-15 tasks total. If fewer than 4, the spec may be too small. If more than 15, consider splitting the spec.

### Phase Rules

- **Phase 1: Setup** — scaffolding, config, no business logic. No dependencies on other tasks.
- **Phase 2: Core** — main business logic, models, services. May depend on Setup tasks.
- **Phase 3: Integration** — wiring components together, API/CLI, integration tests. Depends on Core.
- **Phase 4: Polish** — error handling, edge cases, documentation, cleanup. Depends on Integration.

Not all phases are required. Skip empty phases. Add custom phases if the feature warrants it (e.g., "Phase 3: Migration" for data migration work).

## Step 6: Report

After creating the tasks, report:
- The spec it belongs to (SPEC-NNN, feature name)
- Total number of tasks across phases
- Task dependency graph summary (which tasks block others)
- Estimated parallelization opportunities
- Remind the user: **review the tasks, then run `/execute SPEC-NNN` to start implementation**

## Important Rules

1. **Always use `ctxpin` CLI** to create notes — never create files manually
2. **Always add the `SPEC-NNN` label** to the Tasks note
3. **Read Requirements + Design first** — tasks must trace back to both documents
4. **Read the codebase** — reference real file paths, not hypothetical ones
5. **Only create the Tasks note** — do NOT create or modify Requirements or Design
6. **Every task must be actionable** — no vague items like "implement the feature"
