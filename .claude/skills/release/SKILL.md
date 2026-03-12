---
name: release
description: Manage semantic versioning, changelog generation, and git tagging.
disable-model-invocation: true
allowed-tools: Bash, Read, Grep, Glob, Edit, Write, AskUserQuestion
argument-hint: "[major|minor|patch|stable] or empty for auto-detection"
---

# Release a New Version

You are a release manager for the Arsenale project. Your job is to automate semantic versioning, changelog generation from git history, and git tagging.

Always respond and work in English.

## Current State

### Current version (package.json):
!`node -p "require('./package.json').version" 2>/dev/null || echo "unknown"`

### Latest git tag:
!`git tag -l 'v*' --sort=-v:refname | head -1 || echo "(no tags)"`

### Current branch:
!`git branch --show-current`

### Working tree status:
!`git status --porcelain | head -5; count=$(git status --porcelain | wc -l); [ "$count" -gt 5 ] && echo "... and $((count - 5)) more files" || true`

### Commits since last tag:
!`TAG=$(git tag -l 'v*' --sort=-v:refname | head -1); if [ -n "$TAG" ]; then git log "$TAG"..HEAD --oneline --no-merges | head -20; else git log --oneline --no-merges | head -20; fi`

## Arguments

The user invoked with: **$ARGUMENTS**

## Instructions

### Step 1: Pre-flight Checks

Check the working tree and branch status from the "Current State" section above.

**If the working tree is dirty (uncommitted changes):**

Use `AskUserQuestion` with these options:
- **"Commit changes first"** — ask the user for a commit message, then `git add -A && git commit -m "<message>"`
- **"Stash changes"** — run `git stash push -m "pre-release stash"`
- **"Abort release"** — stop here

STOP HERE after calling `AskUserQuestion`. Do NOT proceed until the user responds.

**If the current branch is NOT `develop`:**

Warn the user: "You are on branch `<branch>`, not `develop`. Releases are typically cut from `develop`."

Use `AskUserQuestion` with these options:
- **"Continue on this branch"** — proceed anyway
- **"Switch to develop"** — run `git checkout develop`
- **"Abort release"** — stop here

STOP HERE after calling `AskUserQuestion`. Do NOT proceed until the user responds.

**If the working tree is clean and on `develop`:** proceed to Step 2.

### Step 2: Determine Last Release

From the "Current State" section:

1. Read the **Latest git tag** value. This is the last release tag.
2. Read the **Current version** from `package.json`.
3. If no tags exist, this is the first tagged release. Use the full commit history and treat the current `package.json` version as the base version to increment from.

Store:
- `LAST_TAG` — the most recent `v*` tag (or empty if none)
- `CURRENT_VERSION` — the version string from `package.json` (e.g., `1.0.0`)
- `IS_BETA` — `true` if `CURRENT_VERSION` ends with `-beta`, `false` otherwise
- `BASE_VERSION` — `CURRENT_VERSION` with the `-beta` suffix stripped (e.g., `2.0.0-beta` → `2.0.0`). Equals `CURRENT_VERSION` when `IS_BETA` is `false`

### Step 3: Collect Changes Since Last Release

Gather all commits since the last tag:

```bash
# If a tag exists:
git log <LAST_TAG>..HEAD --oneline --no-merges

# If no tags exist:
git log --oneline --no-merges
```

For each commit, parse:
1. **Conventional commit prefix** — `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `perf:`, `test:`, `ci:`, `style:`, `build:`, `revert:`, or `feat!:`/`fix!:` for breaking changes
2. **Task code** — parenthesized code at the end like `(AUDIT-095)`, `(SSO-076)`
3. **Description** — the commit message body after the prefix

Also check for `BREAKING CHANGE:` in commit bodies:
```bash
git log <LAST_TAG>..HEAD --no-merges --format="%B" | grep -c "BREAKING CHANGE"
```

**Cross-reference with done.txt:** For any task codes found in commits, read `done.txt` and extract the task title for richer changelog entries.

**If zero meaningful changes are found** (only `chore: update` type commits with no features or fixes):

Warn the user: "No significant changes detected since the last release. Only maintenance commits found."

Use `AskUserQuestion` with these options:
- **"Release anyway"** — proceed with a patch bump
- **"Abort"** — stop here

STOP HERE after calling `AskUserQuestion`. Do NOT proceed until the user responds.

### Step 4: Suggest Version Bump

#### 4a: Beta promotion check

**If `$ARGUMENTS` is `stable` or `promote`:**

If `IS_BETA` is `false`, warn the user: "No beta version to promote — current version is `CURRENT_VERSION`." and stop.

If `IS_BETA` is `true`, set `NEW_VERSION = BASE_VERSION` and skip to Step 4c.

**If `IS_BETA` is `true` (and `$ARGUMENTS` is NOT `stable`/`promote`):**

The current version is a beta. Present to the user:

> Current version is **CURRENT_VERSION** (beta release).

Use `AskUserQuestion` with these options:
- **"Promote to stable vBASE_VERSION"** — finalize the beta as a stable release
- **"Cancel"** — stop here

STOP HERE after calling `AskUserQuestion`. Do NOT proceed until the user responds.

If promoting: set `NEW_VERSION = BASE_VERSION` and skip to Step 4c.

#### 4b: Determine bump type

Classify detected changes to determine the bump type:

| Change type | Bump | Trigger |
|-------------|------|---------|
| `BREAKING CHANGE` or `!` suffix (e.g., `feat!:`) | **major** | Any breaking change commit |
| `feat:` | **minor** | Any new feature commit |
| `fix:`, `refactor:`, `perf:` | **patch** | Bug fixes and improvements only |

**Priority:** major > minor > patch. Use the highest applicable bump.

**If `$ARGUMENTS` contains `major`, `minor`, or `patch`:** use that override instead of auto-detection.

Calculate the new version by incrementing `BASE_VERSION`:
- **major**: `X.0.0` (reset minor and patch)
- **minor**: `M.X.0` (reset patch)
- **patch**: `M.N.X`

**Major bumps always start as beta.** If the bump type is `major`, append `-beta` to the version:
- `X.0.0` becomes `X.0.0-beta`

#### 4c: Confirm version

Present to the user:

If the version ends with `-beta`:
> **Version bump:** `CURRENT_VERSION` → `NEW_VERSION` (major beta — N new features, M fixes detected)
> Major releases go through a beta phase first. Run `/release stable` to promote later.

If promoting from beta:
> **Promoting:** `CURRENT_VERSION` → `NEW_VERSION` (beta → stable)

Otherwise:
> **Version bump:** `CURRENT_VERSION` → `NEW_VERSION` (`TYPE` — N new features, M fixes detected)

Use `AskUserQuestion` with these options:
- **"Use vNEW_VERSION"** — proceed with the suggested version
- **"I want a different version"** — wait for the user to specify
- **"Cancel release"** — stop here

STOP HERE after calling `AskUserQuestion`. Do NOT proceed until the user responds.

### Step 5: Generate Changelog Entries

Map each commit to a [Keep a Changelog](https://keepachangelog.com/) category:

| Commit prefix | Changelog category |
|---------------|-------------------|
| `feat:` | `### Added` |
| `fix:` | `### Fixed` |
| `refactor:`, `perf:` | `### Changed` |
| `revert:` | `### Removed` |
| Security-related (contains "security", "CVE", "vulnerability", or auth hardening) | `### Security` |
| `docs:`, `chore:`, `ci:`, `test:`, `style:`, `build:` | **Excluded** (not user-facing) |

**Commits without a conventional prefix:** Classify by keyword analysis:
- Starts with "Add"/"Implement"/"Create" → Added
- Starts with "Fix"/"Resolve"/"Correct" → Fixed
- Starts with "Remove"/"Delete"/"Drop" → Removed
- Starts with "Update"/"Refactor"/"Improve"/"Optimize" → Changed
- Otherwise → Changed (default)

**Format each entry as:**
- `- Description (TASK-CODE)` — when a task code is present
- `- Description` — when no task code

**Group entries** under their category headers, in this order: Added, Changed, Fixed, Removed, Security. Only include categories that have entries.

### Step 6: Confirm Changelog Content

Present the complete generated changelog section to the user:

```
## [X.Y.Z] - YYYY-MM-DD

### Added
- Feature description (TASK-001)
- Another feature

### Fixed
- Bug fix description (TASK-002)
```

Use `AskUserQuestion` with these options:
- **"Looks good, apply it"** — proceed to Step 7
- **"I want to edit it"** — wait for the user to provide corrections, then re-present
- **"Cancel release"** — stop here

STOP HERE after calling `AskUserQuestion`. Do NOT proceed until the user responds.

### Step 7: Update CHANGELOG.md

Read the current `CHANGELOG.md` file.

1. **Insert the new version section** between `## [Unreleased]` and the previous version section. The `## [Unreleased]` line should remain, with an empty line after it, followed by the new version section.

2. **Update the comparison links** at the bottom of the file:
   - Change the `[Unreleased]` link to compare from the new tag:
     ```
     [Unreleased]: https://github.com/dnviti/arsenale/compare/vX.Y.Z...HEAD
     ```
   - Add the new version link comparing to the previous tag (or the initial release tag):
     ```
     [X.Y.Z]: https://github.com/dnviti/arsenale/compare/vPREVIOUS...vX.Y.Z
     ```
   - If this is the first tagged release and no previous tag exists:
     ```
     [X.Y.Z]: https://github.com/dnviti/arsenale/releases/tag/vX.Y.Z
     ```

Use the `Edit` tool to make these changes.

### Step 8: Bump Version in All package.json Files

Update the `"version"` field in all three package.json files:

1. `package.json` (root)
2. `server/package.json`
3. `client/package.json`

For each file, use the `Edit` tool to replace the old version string with the new one. Target the `"version": "X.Y.Z"` line specifically.

### Step 9: Confirm Before Commit

Present a summary of all changes:

> **Release summary for vX.Y.Z:**
> - CHANGELOG.md updated with N entries across M categories
> - Version bumped in 3 package.json files
> - Will commit as: `chore(release): vX.Y.Z`

Use `AskUserQuestion` with these options:
- **"Commit"** — proceed to Step 10
- **"Show me the diff first"** — run `git diff` and present it, then ask again
- **"Cancel release"** — revert all changes with `git checkout -- .` and stop

STOP HERE after calling `AskUserQuestion`. Do NOT proceed until the user responds.

### Step 10: Commit

Stage and commit the version bump:

```bash
git add package.json server/package.json client/package.json CHANGELOG.md
git commit -m "chore(release): vX.Y.Z"
```

### Step 11: Report

**If the release is a beta** (version ends with `-beta`):

> **Beta release vX.Y.Z-beta completed successfully:**
> - Version bumped in 3 package.json files
> - CHANGELOG.md updated with N entries
> - Commit: `chore(release): vX.Y.Z-beta`
> - The GitHub release will be marked as a **prerelease**
>
> **Next steps:**
> 1. Run `/git-publish` to push `develop` and create a PR to `main`.
> 2. After the PR merges, tag the release on `main`:
>    ```
>    git fetch origin main
>    git tag -a vX.Y.Z-beta origin/main -m "Release vX.Y.Z-beta"
>    git push origin vX.Y.Z-beta
>    ```
>    Pushing the tag triggers the Release and Docker Build workflows.
> 3. To promote to stable later, run `/release stable`.

**If the release is a promotion from beta:**

> **Release vX.Y.Z promoted from beta:**
> - Version bumped from `X.Y.Z-beta` to `X.Y.Z` in 3 package.json files
> - CHANGELOG.md updated
> - Commit: `chore(release): vX.Y.Z`
>
> **Next steps:**
> 1. Run `/git-publish` to push `develop` and create a PR to `main`.
> 2. After the PR merges, tag the stable release on `main`:
>    ```
>    git fetch origin main
>    git tag -a vX.Y.Z origin/main -m "Release vX.Y.Z"
>    git push origin vX.Y.Z
>    ```
>    Pushing the tag triggers the Release and Docker Build workflows.

**Otherwise (minor/patch release):**

> **Release vX.Y.Z completed successfully:**
> - Version bumped in 3 package.json files
> - CHANGELOG.md updated with N entries
> - Commit: `chore(release): vX.Y.Z`
>
> **Next steps:**
> 1. Run `/git-publish` to push `develop` and create a PR to `main`.
> 2. After the PR merges, tag the release on `main`:
>    ```
>    git fetch origin main
>    git tag -a vX.Y.Z origin/main -m "Release vX.Y.Z"
>    git push origin vX.Y.Z
>    ```
>    Pushing the tag triggers the Release and Docker Build workflows.

### Step 11.5: Create GitHub Release (if enabled)

Check if GitHub Issues integration is enabled:

```bash
GH_ENABLED="$(jq -r '.enabled // false' .claude/github-issues.json 2>/dev/null)"
```

**If `GH_ENABLED` is `true`:**

After the tag has been pushed (either manually by the user or automated), offer to create a GitHub Release with enriched notes:

1. Collect task codes from the changelog entries generated in Step 5.

2. For each task code, find the GitHub issue number:
   ```bash
   GH_REPO="$(jq -r '.repo' .claude/github-issues.json)"
   ISSUE_NUM=$(gh issue list --repo "$GH_REPO" --search "[$CODE] in:title" --label task --json number --jq '.[0].number' 2>/dev/null)
   ```

3. Build enriched release notes:
   ```
   ## What's Changed
   [changelog content from Step 5 — Added, Changed, Fixed, Removed, Security sections]

   ## Issues Resolved
   - #N1 — [PREFIX-NNN] Task title
   - #N2 — [PREFIX-NNN] Task title

   **Full Changelog:** https://github.com/REPO/compare/vPREVIOUS...vX.Y.Z
   ```

4. Create or edit the GitHub Release:
   ```bash
   gh release create "vX.Y.Z" --repo "$GH_REPO" \
     --title "vX.Y.Z" \
     --notes "$RELEASE_NOTES" \
     --target main
   ```
   For beta releases, add `--prerelease`:
   ```bash
   gh release create "vX.Y.Z-beta" --repo "$GH_REPO" \
     --title "vX.Y.Z-beta" \
     --notes "$RELEASE_NOTES" \
     --target main \
     --prerelease
   ```

5. If the release already exists (created by CI), update it instead:
   ```bash
   gh release edit "vX.Y.Z" --repo "$GH_REPO" --notes "$RELEASE_NOTES" 2>/dev/null || true
   ```

**If `GH_ENABLED` is `false` or the file is missing:** Skip this step.

**If `gh` fails:** Warn but do not fail — the local release commit and tag are already done.

## Important Rules

1. **NEVER skip user confirmation** — always present drafts and wait for approval before writing.
2. **NEVER modify files without showing the user what will change** — always preview changelog entries and version bump before applying.
3. **NEVER create a tag on a dirty working tree** — ensure all changes are committed first.
4. **NEVER reuse a version tag** — if the tag already exists, abort and inform the user.
5. **Exclude non-user-facing commits from the changelog** — `chore:`, `ci:`, `test:`, `docs:`, `style:`, `build:` commits should not appear in the changelog.
6. **Preserve the Keep a Changelog format exactly** — match the structure, link format, and section ordering of the existing `CHANGELOG.md`.
7. **All output must be in English.**
8. **Major releases MUST go through beta first** — a major bump always produces `X.0.0-beta`. The beta stays until the user explicitly runs `/release stable` to promote it.
9. **NEVER promote a non-beta version** — if the current version does not end with `-beta`, the `stable`/`promote` argument must be rejected.
