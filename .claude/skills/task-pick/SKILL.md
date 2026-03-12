---
name: task-pick
description: Pick up the next status:todo task for implementation.
disable-model-invocation: true
argument-hint: "[TASK-CODE]"
---

# Pick Up a Task

You are a task manager for the Arsenale project. Your job is to:
1. Pick up a `status:todo` task and begin implementation

## Mode Detection

Determine the operating mode first:

```bash
TRACKER_CFG=".claude/issues-tracker.json"; [ ! -f "$TRACKER_CFG" ] && TRACKER_CFG=".claude/github-issues.json"
PLATFORM="$(jq -r '.platform // "github"' "$TRACKER_CFG" 2>/dev/null)"
TRACKER_ENABLED="$(jq -r '.enabled // false' "$TRACKER_CFG" 2>/dev/null)"
TRACKER_SYNC="$(jq -r '.sync // false' "$TRACKER_CFG" 2>/dev/null)"
TRACKER_REPO="$(jq -r '.repo' "$TRACKER_CFG" 2>/dev/null)"
```

- **Platform-only mode** (`TRACKER_ENABLED=true` AND `TRACKER_SYNC != true`): Read/write task state via GitHub Issues. No local file operations.
- **Dual sync mode** (`TRACKER_ENABLED=true` AND `TRACKER_SYNC=true`): Use local files as primary, then sync to GitHub.
- **Local only mode** (`TRACKER_ENABLED=false` or config missing): Use local files only.

## Current Task State

### GitHub-only mode:

```bash
# Pending tasks (by priority)
gh issue list --repo "$TRACKER_REPO" --label "task,status:todo,priority:high" --state open --json number,title --jq '.[] | "\(.title)"' 2>/dev/null
gh issue list --repo "$TRACKER_REPO" --label "task,status:todo,priority:medium" --state open --json number,title --jq '.[] | "\(.title)"' 2>/dev/null
gh issue list --repo "$TRACKER_REPO" --label "task,status:todo,priority:low" --state open --json number,title --jq '.[] | "\(.title)"' 2>/dev/null
# Completed tasks
gh issue list --repo "$TRACKER_REPO" --label "task,status:done" --state closed --limit 20 --json number,title --jq '.[] | "\(.title)"' 2>/dev/null
```

### Local/Dual mode:

#### Pending tasks (from to-do.txt):
!`grep '^\[ \]' to-do.txt | tr -d '\r'`

#### Completed tasks (from done.txt):
!`grep '^\[x\]' done.txt 2>/dev/null | tr -d '\r'`

#### Recommended implementation order:
!`grep -A 50 'ORDINE DI IMPLEMENTAZIONE CONSIGLIATO' to-do.txt 2>/dev/null | tr -d '\r'`

## Instructions

The user wants to pick up a task. The argument provided is: **$ARGUMENTS**

---

### Step 1: Determine which task to pick

**In GitHub-only mode:**
- **If a task code was provided** (e.g., `CRED-006`): Search for it: `gh issue list --repo "$TRACKER_REPO" --search "[TASK-CODE] in:title" --label "task,status:todo" --state open --json number,title`
  - If not found in todo, check if already done: `gh issue list --repo "$TRACKER_REPO" --search "[TASK-CODE] in:title" --label "task,status:done" --state closed --json number,title`
  - If done, inform the user and suggest next available task.
- **If no argument was provided**: Select the next task by priority label ordering: `priority:high` first, then `priority:medium`, then `priority:low`. Within same priority, pick the lowest-numbered task. Check dependencies by reading the task body — dependency task codes should have `status:done` label.

**In local/dual mode:**
- **If a task code was provided**: Verify it exists in `to-do.txt` as `[ ]`. If in `done.txt` as `[x]`, inform the user.
- **If no argument was provided**: Select from the recommended implementation order that is still `[ ]` in `to-do.txt`. Skip completed or in-progress tasks. Verify dependencies are satisfied.

### Step 2: Mark task as in-progress

**In GitHub-only mode:**

Update the GitHub Issue labels:
```bash
ISSUE_NUM=$(gh issue list --repo "$TRACKER_REPO" --search "[TASK-CODE] in:title" --label "task,status:todo" --state open --json number --jq '.[0].number')
gh issue edit "$ISSUE_NUM" --repo "$TRACKER_REPO" --remove-label "status:todo" --add-label "status:in-progress"
gh issue comment "$ISSUE_NUM" --repo "$TRACKER_REPO" --body "Task picked up. Branch: \`task/<task-code-lowercase>\`"
```

**In dual sync mode:**

1. Read the full task block from `to-do.txt` (between `------` separators, inclusive).
2. Remove that entire block from `to-do.txt`.
3. Append the block to `progressing.txt`.
4. Change `[ ]` to `[~]` in the header line.
5. Update recommended order annotation to `[IN CORSO]` if applicable.
6. Sync to GitHub (update labels as above).

**In local only mode:**

Same as dual sync steps 1-5, skip GitHub sync.

### Step 2.5: Create a task branch

Create a dedicated git branch for this task, branching from `develop`.

**2.5a. Check the working tree:**
```bash
git status --porcelain
```
If dirty, inform the user and stop.

**2.5b. Switch to develop and pull latest:**
```bash
git checkout develop
git pull origin develop
```

**2.5c. Create the task branch:**
```bash
git branch --list "task/<task-code-lowercase>"
```
- If exists: `git checkout task/<task-code-lowercase>`
- If not: `git checkout -b task/<task-code-lowercase>`

### Step 3: Read the full task details

**In GitHub-only mode:**
- `gh issue view $ISSUE_NUM --repo "$TRACKER_REPO" --json body --jq '.body'`
- Parse the structured body: Description, Technical Details, Files Involved sections.

**In local/dual mode:**
- Read the complete task block from `progressing.txt`.

### Step 4: Explore the codebase

For each file listed in the files involved section:
- If the file exists, read it to understand the current state
- If marked to create, check the target directory and look at similar files for patterns
- Identify relevant interfaces, types, and patterns

### Step 5: Present the implementation briefing

Present a clear English-language briefing:

1. **Task Selected**: Code, title, and priority
2. **Status Update**: Confirm the task was marked as in-progress
3. **Scope Summary**: What needs to be done
4. **Technical Approach**: Implementation steps based on task details and codebase exploration
5. **Files to Create/Modify**: Every file with what needs to happen in each
6. **Dependencies**: Status of all dependencies
7. **Risks**: Any concerns found during exploration
8. **Prisma Migration**: Note if schema changes are involved
9. **Quality Gate**: Remind that `npm run verify` must pass before closing

After presenting the briefing, ask the user: "Ready to start implementation, or would you like to adjust the approach?"

---

### Step 6: Post-Implementation — Confirm, Close & Commit

After a task has been **fully implemented and the quality gate (`npm run verify`) passes**, execute this completion flow:

**6a. Present a Testing Guide:**

Generate and present a **manual testing guide** derived from the task's technical details and files involved.

Format:
> ### Testing Guide for [TASK-CODE] — [Task Title]
>
> **Prerequisites:**
> - [What needs to be running]
>
> **Steps to test:**
> 1. [Concrete action]
>    - **Expected:** [Result]
>
> **Edge cases to check:**
> - [2-3 edge cases]

The guide must be actionable and specific — use real URLs, UI element names, and API endpoints.

**6a.5. Mark task as to-test (if platform integration is enabled):**

If `TRACKER_ENABLED` is `true`:
```bash
ISSUE_NUM=$(gh issue list --repo "$TRACKER_REPO" --search "[TASK-CODE] in:title" --label task --state open --json number --jq '.[0].number')
gh issue edit "$ISSUE_NUM" --repo "$TRACKER_REPO" --add-label "status:to-test"
```

**6b. Ask for user confirmation:**

Use `AskUserQuestion` with options:
- **"Yes, task is done"** — proceed to 6b.5 then 6c
- **"Not yet, needs more work"** — stop; task stays in-progress
- **"Skip testing, mark as done"** — skip to 6c directly (to-test label remains for later verification)

**6b.5. Remove to-test label (if platform integration is enabled):**

If `TRACKER_ENABLED` is `true` and the user confirmed testing:
```bash
gh issue edit "$ISSUE_NUM" --repo "$TRACKER_REPO" --remove-label "status:to-test"
```

**6c. Mark task as done:**

**In GitHub-only mode:**
```bash
ISSUE_NUM=$(gh issue list --repo "$TRACKER_REPO" --search "[TASK-CODE] in:title" --label task --state open --json number --jq '.[0].number')
gh issue edit "$ISSUE_NUM" --repo "$TRACKER_REPO" --remove-label "status:in-progress" --add-label "status:done"
gh issue close "$ISSUE_NUM" --repo "$TRACKER_REPO" --comment "Task completed and verified. Quality gate passed."
```

**In dual sync mode:**
1. Read the full task block from `progressing.txt` (between `------` separators, inclusive)
2. Remove it from `progressing.txt`
3. Append to `done.txt` in the appropriate section
4. Change `[~]` to `[x]` in the header line
5. Add a `COMPLETATO:` line with a brief English summary
6. Update recommended order annotation to `[COMPLETATO]` if applicable
7. Sync to GitHub (update labels and close issue as above)

**In local only mode:**
Same as dual sync steps 1-6, skip GitHub sync.

Inform the user: "Task [TASK-CODE] has been closed."

**6d. Ask to commit:**

Use `AskUserQuestion` with options:
- **"Yes, commit"** — create a commit referencing the task code
- **"No, skip commit"** — skip

**6e. Ask to merge into develop:**

Use `AskUserQuestion` with options:
- **"Yes, merge into develop"** — execute:
  ```bash
  git checkout develop
  git merge task/<task-code-lowercase> --no-ff -m "Merge task/<task-code-lowercase> into develop"
  ```
  Use `--no-ff` to preserve branch history.

  **Note:** If the task still has `status:to-test` label (user skipped testing), warn: "This task has not been tested yet. Consider running `/test-engineer TASK-CODE` before merging to a release branch."

- **"No, stay on task branch"** — skip the merge

**Important:** Always ask — never auto-commit, auto-close, or auto-merge without user confirmation.
