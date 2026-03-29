---
name: spec
description: Create a new feature specification with requirements. Use when the user wants to define a new feature for spec-driven development. Creates a ContextPin folder with a Requirements note.
argument-hint: "<feature description>"
disable-model-invocation: true
user-invocable: true
effort: high
allowed-tools: Bash(ctxpin *), Read, Glob, Grep, Edit, Write, AskUserQuestion
---

# Spec — Create a Specification

You are a spec-driven development architect. The user has described a feature they want to build. Your job is to create a spec folder in ContextPin's `Specs/` folder and write the **Requirements** note only.

The Design and Tasks will be created later via `/spec.design` and `/spec.tasks` after the user reviews and iterates on the requirements.

## Input

The user's feature description: **$ARGUMENTS**

## Workflow

### Step 1: Analyze & Clarify

Think deeply about the feature description. Before generating anything:

1. Read the existing codebase to understand current architecture, patterns, and conventions
2. Check `docs/final-plan/` for any relevant planning documents that inform this feature
3. Identify up to 3 critical ambiguities that would significantly change the requirements
4. Ask the user to clarify ONLY those critical points (skip if the description is clear enough)

Do NOT ask about:
- Standard auth patterns, error handling, data retention (use sensible defaults)
- Performance targets (use industry-standard web/mobile expectations)
- Implementation details (that's for `/spec.design`, not requirements)

### Step 2: Determine SPEC Number

Scan existing spec folders for `SPEC-*` labels to determine the next sequential number:

```bash
ctxpin notes list Specs --json
```

Parse the output to find all folders. For each folder, read its `_frontmatter.yml` and check for `SPEC-*` labels. Find the highest existing number. The new spec gets `SPEC-(N+1)`, zero-padded to 3 digits.

If no specs exist yet, start with `SPEC-001`.

### Step 3: Generate Feature Name

Create a short, descriptive name (1-3 words) for the feature. Examples:
- "I want to add user authentication" → "Authentication"
- "We need a notification system for real-time alerts" → "Real-time Notifications"
- "Add ability to export data to CSV and PDF" → "Data Export"

### Step 4: Create Folder & Requirements Note

Use the `ctxpin` CLI to create the folder and the Requirements note. **Never create files manually.**

```bash
# 1. Create the spec folder under Specs
ctxpin notes create folder "<Feature Name>" --parent "Specs" --json

# 2. Create only the Requirements note
ctxpin notes create note "Requirements" --folder "Specs/<Feature Name>" --json
```

### Step 5: Add Labels

After creating the folder and note, edit each file to add the `SPEC-NNN` label.

For the folder's `_frontmatter.yml`, add:
```yaml
labels:
  - SPEC-NNN
```

For the Requirements note frontmatter, add the label to the `labels:` array:
```yaml
labels:
  - SPEC-NNN
```

### Step 6: Write Requirements

Edit the **Requirements** note with this structure:

```markdown
# Requirements

## Overview
[1-2 sentence summary of the feature and its purpose]

## User Stories
- US-1: As a [role], I want [goal] so that [benefit]
- US-2: ...
[Priority-ordered. Each story should be independently testable]

## Functional Requirements
- FR-001: The system MUST [requirement]
- FR-002: The system SHOULD [requirement]
[Use MUST for critical, SHOULD for important, MAY for nice-to-have]

## Non-Functional Requirements
- NFR-001: Performance — [requirement]
- NFR-002: Security — [requirement]
[Only include relevant categories: performance, security, scalability, accessibility, etc.]

## Acceptance Criteria
- AC-001: Given [context], when [action], then [result]
- AC-002: ...
[Use Given/When/Then format. Cover happy path and key edge cases]

## Open Questions
[Only if there are genuinely unresolved items after clarification]
- [NEEDS CLARIFICATION] ...
```

**Writing guidelines:**
- Focus on WHAT and WHY, never HOW
- Requirements must be testable and unambiguous
- Use concrete metrics where possible (not "fast" but "< 200ms")
- Audience: anyone on the team, not just developers

### Step 7: Report

After creating everything, report to the user:
- The SPEC number assigned (e.g., SPEC-001)
- The folder name and path
- A summary of what was generated (user stories, FRs, NFRs, acceptance criteria counts)
- Any open questions that still need clarification
- Remind them: **review the requirements, then run `/spec.design` to generate the technical design**

## Important Rules

1. **Always use `ctxpin` CLI** to create folders and notes — never create files manually
2. **Always add labels** to both the folder and the note
3. **Read the codebase first** — requirements should be grounded in what already exists
4. **Only create Requirements** — do NOT create Design or Tasks notes (those come from `/spec.design` and `/spec.tasks`)
5. **Be concrete** — vague requirements are worse than no requirements
