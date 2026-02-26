---
name: task-pick
description: Pick up the next task for implementation (or a specific task by code). Updates to-do.txt and sets up implementation context.
disable-model-invocation: true
argument-hint: "[TASK-CODE]"
allowed-tools: Bash, Read, Grep, Glob, Edit, Write
---

# Pick Up a Task

You are a task manager for the Remote Desktop Manager project. Your job is to pick up a task, update its status, and prepare full implementation context.

## Current Task State

### Pending tasks (available to pick up):
!`grep '^\[ \]' to-do.txt | tr -d '\r'`

### Already in-progress tasks:
!`grep '^\[~\]' to-do.txt | tr -d '\r'`

### Completed tasks:
!`grep '^\[x\]' to-do.txt | tr -d '\r'`

### Recommended implementation order:
!`sed -n '/ORDINE DI IMPLEMENTAZIONE CONSIGLIATO/,/NOTE/p' to-do.txt | head -30 | tr -d '\r'`

## Instructions

The user wants to pick up a task. The argument provided is: **$ARGUMENTS**

### Step 1: Determine which task to pick

- **If a task code was provided** (e.g., `SHR-005`): Use that specific task. Verify it exists in `to-do.txt` and is in `[ ]` (todo) status. If already `[~]` (in-progress), inform the user and show its details without changing anything. If `[x]` (completed), inform the user and suggest the next available task.

- **If no argument was provided**: Select the next task from the recommended implementation order that is still `[ ]` (todo). Skip completed `[x]` or in-progress `[~]` tasks. Also verify that the task's dependencies are satisfied (dependency tasks should be `[x]` completed). If a task has unsatisfied dependencies, skip it and pick the next one.

### Step 2: Update to-do.txt

Update the selected task's status from `[ ]` to `[~]` in `to-do.txt`. Use a precise edit that changes only the status marker on the correct line. Example:

Change:
```
[ ] SHR-005 — Condivisione connessioni dal menu contestuale (view/edit)
```
to:
```
[~] SHR-005 — Condivisione connessioni dal menu contestuale (view/edit)
```

**Important:** Only change the status marker. Do not alter any other content.

### Step 3: Read the full task details

Read the complete task block from `to-do.txt` for the selected task — everything between its `------` separator lines: priority, dependencies, description, technical details, and files involved.

### Step 4: Explore the codebase

For each file listed in the "FILE COINVOLTI" (files involved) section:
- If the file exists, read it to understand the current state
- If marked "CREARE" (to create), check the target directory and look at similar files for patterns to follow
- Identify relevant interfaces, types, and patterns

### Step 5: Present the implementation briefing

Present a clear English-language briefing:

1. **Task Selected**: Code, title, and priority
2. **Status Update**: Confirm the task was marked as in-progress in to-do.txt
3. **Scope Summary**: What needs to be done (in English)
4. **Technical Approach**: Implementation steps based on task details and codebase exploration
5. **Files to Create/Modify**: Every file with what needs to happen in each
6. **Dependencies**: Status of all dependencies
7. **Risks**: Any concerns found during exploration

After presenting the briefing, ask the user: "Ready to start implementation, or would you like to adjust the approach?"
