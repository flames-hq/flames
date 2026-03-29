---
name: spec.execute
description: Execute implementation tasks from a spec. Reads tasks from ContextPin spec notes and implements them with full context from Requirements and Design documents. Run after /spec.tasks.
argument-hint: "[SPEC-NNN] [TASK-NNN ...]"
disable-model-invocation: true
user-invocable: true
effort: high
allowed-tools: Bash, Read, Glob, Grep, Edit, Write, EnterPlanMode, ExitPlanMode, Agent
---

# Execute — Implement Tasks from a Spec

You are an implementation agent for spec-driven development. Your job is to execute tasks defined in a ContextPin spec, following the Requirements and Design documents for context.

## Input

Arguments: **$ARGUMENTS**

## Step 1: Parse Arguments & Resolve Spec

Parse the arguments to determine what to execute:

**Pattern detection:**
- Arguments starting with `SPEC-` → explicit spec identifier
- Arguments starting with `TASK-` → specific task IDs
- No arguments → infer from context or ask

**Resolution rules:**

1. **`SPEC-NNN` provided** → Use that spec. Any remaining `TASK-NNN` args filter to those specific tasks.
2. **Only `TASK-NNN` args** → Infer the spec from the active ContextPin context (look for `SPEC-*` labels in the current context injected by the ADE hook). If no context, list specs and ask.
3. **No arguments** → Check if there's an active ContextPin document in context with a `SPEC-*` label. If yes, use that spec and execute all pending tasks. If no, list available specs and ask which to execute.

**To list specs:**
```bash
ctxpin notes list Specs --json
```

### Resolve Spec Folder

Once you have the `SPEC-NNN` identifier:
1. List folders in `Specs/` and find the one whose `_frontmatter.yml` contains the matching `SPEC-NNN` label
2. Read the three notes from that folder: Requirements, Design, Tasks

## Step 2: Load Context

Read all three spec documents to understand the full picture:

1. **Requirements** — understand what needs to be built (user stories, FRs, acceptance criteria)
2. **Design** — understand how it should be built (architecture, data model, interfaces)
3. **Tasks** — the ordered task list with dependencies and phases

Parse the Tasks note to extract:
- All tasks with their IDs, descriptions, phase, dependencies, and completion status
- Which tasks are pending (`- [ ]`) vs completed (`- [x]`)

## Step 3: Determine Execution Scope

Based on arguments:

| Scenario | What to execute |
|----------|----------------|
| All pending tasks | All `- [ ]` items, respecting phase and dependency order |
| Specific TASK-NNN list | Only those tasks, but still respect dependency order |

**Dependency resolution:**
- If TASK-005 depends on TASK-003, and TASK-003 is not yet completed, execute TASK-003 first (warn the user)
- If a dependency is already marked `[x]`, skip it
- Within a phase, execute tasks in order unless marked `[P]` (parallelizable)

## Step 4: Execute Tasks

For each task in the execution queue:

### 4a. Announce
Tell the user which task you're starting:
```
Executing TASK-003 — [task description]
```

### 4b. Plan
Enter plan mode to design the implementation approach for this specific task:
- Reference the Requirements and Design documents
- Identify the specific files to create or modify
- Consider existing codebase patterns and conventions
- Plan tests if applicable (TDD approach)

### 4c. Implement
After the plan is approved:
- Write the code following the plan
- Follow existing codebase conventions
- Write tests where the task calls for it
- Keep changes focused on the task scope — don't refactor unrelated code

### 4d. Mark Complete
After successful implementation, update the Tasks note:
- Change `- [ ] TASK-NNN` to `- [x] TASK-NNN`
- Update the note's `updated_at` timestamp in frontmatter to the current UTC time

### 4e. Report
After each task:
```
Completed TASK-003 — [brief summary of what was done]
Files modified: [list]
```

## Step 5: Final Report

After all tasks in scope are complete, report:
- Total tasks executed
- Tasks remaining (if executing a subset)
- Any issues encountered
- Files created or modified across all tasks

## Execution Rules

1. **Read Requirements + Design first** — always understand the full context before writing any code
2. **Respect dependency order** — never execute a task before its dependencies are complete
3. **One task at a time** — complete and mark each task before moving to the next
4. **Stop on failure** — if a task fails (tests don't pass, blocking issue), stop and report rather than continuing
5. **Stay in scope** — implement exactly what the task describes, nothing more
6. **Use plan mode** — for non-trivial tasks, enter plan mode to get user alignment before implementing
7. **Update the Tasks note** — always mark tasks `[x]` after completion so progress is tracked

## Error Handling

- **Spec not found**: List available specs and ask the user which one
- **Task not found**: Show available tasks in the spec and ask which to execute
- **Dependency not met**: Warn and offer to execute the dependency first
- **Implementation failure**: Stop, report the error, suggest next steps
- **Ambiguous context**: If multiple specs are in context, ask the user to specify
