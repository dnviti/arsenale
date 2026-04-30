# Fix PR #602 blockers and CI failures

## TL;DR
> **Summary**: Fix the confirmed PR regressions in shared secret payload rendering, correct the Docker/Gateway publish policy so `develop` emits `latest` and `main` emits `stable` only, and clear the current lint error backlog that is blocking PR #602 checks.
> **Deliverables**:
> - Shared secret renderer updated for all required payload fields used by `PublicSharePage`
> - Docker and gateway workflows aligned with the documented branch/tag publish policy
> - Repository lint errors reduced from 16 blocking errors to zero without expanding scope into the 96 non-blocking warnings
> - Targeted tests and full verification evidence captured under `.sisyphus/evidence/`
> **Effort**: Medium
> **Parallel**: YES - 2 waves
> **Critical Path**: Tasks 1-5 → Task 6 → Task 7 → Tasks 8-9 → Final Verification Wave

## Context
### Original Request
- Review PR #602, verify the reported issues, then produce a fix plan for each problem.

### Interview Summary
- Confirmed PR-scoped issues:
  - Shared public secret rendering is incomplete relative to the payload union and existing detail renderers.
  - Docker/Gateway workflow publish logic still risks emitting an extra branch-ref tag from `main` even though the PR intent says `main -> stable`.
- Confirmed CI state:
  - The two failing checks (`verify-client` and `Browser Extension / verify`) fail on the same repo-wide lint run.
  - The blocking lint failures are pre-existing and mostly outside the PR diff, but the user explicitly wants them included in scope so PR #602 can go green.
- Explicit scope decision:
  - Include lint-debt cleanup required to restore green checks on the PR branch.

### Metis Review (gaps addressed)
- Metis confirmed three real field gaps in the new shared secret renderer: `LOGIN.domain`, `CERTIFICATE.expiresAt`, and `API_KEY.headers`.
- Metis confirmed the workflow bug: enabling `type=ref,event=branch` for `main` produces an unwanted `:main` tag in addition to `:stable`.
- Metis highlighted that `fetch-depth: 0` is functionally required for the ancestry gate (`git branch -r --contains`) and must remain.
- Metis warned against letting the lint cleanup expand into non-blocking warnings; this plan limits lint work to the 16 current errors.
- Metis flagged `SECURE_NOTE` sensitivity as ambiguous; this plan keeps current behavior because both existing client renderers already show secure note content in clear text.

## Work Objectives
### Core Objective
Land a minimal-risk fix set for PR #602 that resolves the confirmed review issues and restores passing CI without broadening scope into unrelated warnings or workflow redesign.

### Deliverables
- `client/src/components/secrets/SecretPayloadView.tsx` created if absent on the working branch, then updated to render all required shared payload fields.
- `client/src/pages/PublicSharePage.test.tsx` extended to cover the missing payload branches.
- `client/src/components/Dialogs/SettingsDialog.tsx` refactored to eliminate `react-hooks/set-state-in-effect` violations without `eslint-disable` comments.
- `client/src/components/ui/sidebar.tsx` updated to remove impure render-time randomness.
- Mechanical unused-import/unused-variable lint cleanup across the 11 failing files.
- `.github/workflows/docker-build.yml` and `.github/workflows/gateways-build.yml` updated so:
  - `develop` emits `latest`
  - `main` emits `stable` only
  - semver tags emit only for `v*` tags whose commit is on `origin/main`
  - `fetch-depth: 0` remains in place

### Definition of Done (verifiable conditions with commands)
- `npm run lint` exits `0` from repo root.
- `npm run typecheck -w client` exits `0`.
- `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx"` exits `0` from `client/`.
- `npx vitest run "src/pages/PublicSharePage.test.tsx"` exits `0` from `client/`.
- `npx vitest run "src/components/ui/sidebar.test.tsx"` exits `0` from `client/`.
- `npm run verify` exits `0` from repo root.

### Must Have
- Fix only the confirmed missing fields in the shared payload renderer: `domain`, payload-level certificate expiry, and API key headers.
- Keep `SECURE_NOTE` behavior unchanged unless a separate product decision is introduced later.
- Keep `fetch-depth: 0` in both publish workflows.
- Eliminate all 16 current lint errors, including the 4 purity errors, without introducing new ignores for unused imports.
- Preserve reusable verify/security/scan jobs and their existing command contracts.

### Must NOT Have (guardrails, AI slop patterns, scope boundaries)
- Do not touch the 96 current lint warnings in this plan.
- Do not redesign workflow structure, extract reusable workflow helpers, or rewrite the publish model beyond the branch/tag rules above.
- Do not add `eslint-disable` comments for the unused-import or purity issues.
- Do not change `SECURE_NOTE` rendering semantics in the public share flow.
- Do not move `SecretPayloadView` out of `client/src/components/secrets/` as part of this fix set once the file exists on the working branch.

## Verification Strategy
> ZERO HUMAN INTERVENTION - all verification is agent-executed.
- Test decision: tests-after + existing Vitest/ESLint/TypeScript/repo verify workflows.
- QA policy: Every task includes agent-executed scenarios with concrete commands or browser interactions.
- Evidence: `.sisyphus/evidence/task-{N}-{slug}.{ext}`

## Execution Strategy
### Parallel Execution Waves
> Target: 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks for max parallelism.

Wave 1: mechanical lint cleanup + state-derivation refactors + sidebar purity fix (Tasks 1-5)

Wave 2: shared payload rendering fixes + workflow publish-policy alignment (Tasks 6-9)

### Dependency Matrix (full, all tasks)
- Task 1: no blockers
- Task 2: no blockers
- Task 3: no blockers
- Task 4: Blocked by Task 3 (shared state-model refactor in `SettingsDialog.tsx`)
- Task 5: no blockers
- Task 6: no blockers
- Task 7: Blocked by Task 6 (same renderer and shared page test file)
- Task 8: no blockers
- Task 9: no blockers
- Final Verification Wave: Blocked by Tasks 1-9

### Agent Dispatch Summary (wave → task count → categories)
- Wave 1 → 5 tasks → `quick` + `unspecified-low`
- Wave 2 → 4 tasks → `quick`
- Final Verification → 4 tasks → `oracle`, `unspecified-high`, `deep`

## TODOs
> Implementation + Test = ONE task. Never separate.
> EVERY task MUST have: Agent Profile + Parallelization + QA Scenarios.

- [x] 1. Remove unused imports and vars in dialog/keychain files

  **What to do**: Delete the currently unused imports/vars reported by lint in the dialog/keychain cluster only: `client/src/components/Dialogs/ConnectionAuditLogDialog.tsx`, `client/src/components/Dialogs/KeychainDialog.tsx`, `client/src/components/Keychain/ExternalShareDialog.tsx`, `client/src/components/Keychain/PasswordRotationPanel.tsx`, `client/src/components/Keychain/SecretDetailView.tsx`, `client/src/components/Keychain/SecretDialog.tsx`, and `client/src/components/Keychain/ShareSecretDialog.tsx`. Remove the dead symbols instead of renaming them to `_` or adding eslint suppressions.
  **Must NOT do**: Do not touch runtime logic, dialog copy, styles, or any warning-only lint findings in these files.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: mechanical cleanup across a bounded set of files with direct lint pointers
  - Skills: [`tests`] - Reason: preserve targeted verification discipline while editing UI files
  - Omitted: [`playwright`] - Reason: task is fully verifiable with lint/typecheck; no browser interaction required

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: none | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Lint targets: `client/src/components/Dialogs/ConnectionAuditLogDialog.tsx:157`, `client/src/components/Dialogs/KeychainDialog.tsx:31`
  - Lint targets: `client/src/components/Keychain/ExternalShareDialog.tsx:16`, `client/src/components/Keychain/PasswordRotationPanel.tsx:3,14`
  - Lint targets: `client/src/components/Keychain/SecretDetailView.tsx:3`, `client/src/components/Keychain/SecretDialog.tsx:15`, `client/src/components/Keychain/ShareSecretDialog.tsx:15`
  - Pattern: `eslint.config.mjs` - current error policy; unused-vars errors must be removed, not ignored

  **Acceptance Criteria** (agent-executable only):
  - [ ] `npx eslint client/src/components/Dialogs/ConnectionAuditLogDialog.tsx client/src/components/Dialogs/KeychainDialog.tsx client/src/components/Keychain/ExternalShareDialog.tsx client/src/components/Keychain/PasswordRotationPanel.tsx client/src/components/Keychain/SecretDetailView.tsx client/src/components/Keychain/SecretDialog.tsx client/src/components/Keychain/ShareSecretDialog.tsx` exits `0`
  - [ ] `npm run typecheck -w client` exits `0` after the cleanup

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Targeted lint passes for dialog/keychain cleanup
    Tool: Bash
    Steps: Run `npx eslint client/src/components/Dialogs/ConnectionAuditLogDialog.tsx client/src/components/Dialogs/KeychainDialog.tsx client/src/components/Keychain/ExternalShareDialog.tsx client/src/components/Keychain/PasswordRotationPanel.tsx client/src/components/Keychain/SecretDetailView.tsx client/src/components/Keychain/SecretDialog.tsx client/src/components/Keychain/ShareSecretDialog.tsx` from repo root
    Expected: Exit code 0 and no `@typescript-eslint/no-unused-vars` errors for the listed files
    Evidence: .sisyphus/evidence/task-1-dialog-keychain-lint.txt

  Scenario: Cleanup does not break referenced symbols
    Tool: Bash
    Steps: Run `npm run typecheck -w client`
    Expected: Exit code 0 with no new missing-symbol or import errors in the touched files
    Evidence: .sisyphus/evidence/task-1-dialog-keychain-typecheck.txt
  ```

  **Commit**: YES | Message: `lint(client): remove unused dialog and keychain imports` | Files: `[client/src/components/Dialogs/ConnectionAuditLogDialog.tsx, client/src/components/Dialogs/KeychainDialog.tsx, client/src/components/Keychain/ExternalShareDialog.tsx, client/src/components/Keychain/PasswordRotationPanel.tsx, client/src/components/Keychain/SecretDetailView.tsx, client/src/components/Keychain/SecretDialog.tsx, client/src/components/Keychain/ShareSecretDialog.tsx]`

- [x] 2. Remove remaining unused imports and vars in non-dialog client files

  **What to do**: Remove the remaining unused symbols flagged by the current lint run in `client/src/components/DatabaseClient/DbSchemaBrowser.tsx`, `client/src/components/Workspace/CommandPalette.tsx`, `client/src/components/orchestration/GatewayInstanceList.tsx`, and `client/src/pages/LoginPage.test.tsx`.
  **Must NOT do**: Do not change query behavior, command palette UX, orchestration loading states, or test assertions beyond deleting the dead binding.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: small mechanical edits with direct error locations
  - Skills: [`tests`] - Reason: one touched file is a test and should stay green
  - Omitted: [`playwright`] - Reason: command-level verification is sufficient

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: none | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Lint targets: `client/src/components/DatabaseClient/DbSchemaBrowser.tsx:12`, `client/src/components/Workspace/CommandPalette.tsx:10`
  - Lint targets: `client/src/components/orchestration/GatewayInstanceList.tsx:4`, `client/src/pages/LoginPage.test.tsx:287`
  - Test pattern: `client/src/pages/PublicSharePage.test.tsx:16-24` - simple render helper pattern used in current client tests

  **Acceptance Criteria** (agent-executable only):
  - [ ] `npx eslint client/src/components/DatabaseClient/DbSchemaBrowser.tsx client/src/components/Workspace/CommandPalette.tsx client/src/components/orchestration/GatewayInstanceList.tsx client/src/pages/LoginPage.test.tsx` exits `0`
  - [ ] `npm run typecheck -w client` exits `0` after the cleanup

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Targeted lint passes for remaining unused symbols
    Tool: Bash
    Steps: Run `npx eslint client/src/components/DatabaseClient/DbSchemaBrowser.tsx client/src/components/Workspace/CommandPalette.tsx client/src/components/orchestration/GatewayInstanceList.tsx client/src/pages/LoginPage.test.tsx` from repo root
    Expected: Exit code 0 with no unused-var errors in the listed files
    Evidence: .sisyphus/evidence/task-2-misc-lint.txt

  Scenario: Login page tests still compile and run
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/pages/LoginPage.test.tsx"`
    Expected: Exit code 0; the test file passes after removing the dead `view` binding
    Evidence: .sisyphus/evidence/task-2-loginpage-test.txt
  ```

  **Commit**: YES | Message: `lint(client): remove remaining unused vars` | Files: `[client/src/components/DatabaseClient/DbSchemaBrowser.tsx, client/src/components/Workspace/CommandPalette.tsx, client/src/components/orchestration/GatewayInstanceList.tsx, client/src/pages/LoginPage.test.tsx]`

- [x] 3. Derive expanded concern state in `SettingsDialog` instead of mutating it from effects

  **What to do**: Replace the current `expandedConcerns` effect-driven sync with a two-layer model in `client/src/components/Dialogs/SettingsDialog.tsx`: keep only manually toggled/seeded concern ids in state, then derive the effective expanded set via `useMemo` by unioning (a) manual expansions, (b) the active `resolvedConcern`, and (c) every filtered concern while search is non-empty. Remove the two effect blocks at lines `242-260` and update `toggleConcernExpanded`/`handleConcernClick` to mutate only the manual state.
  **Must NOT do**: Do not add `eslint-disable` comments, do not change filtering semantics, and do not break persistence of the resolved concern.

  **Recommended Agent Profile**:
  - Category: `unspecified-low` - Reason: localized behavioral refactor in a single dialog with existing tests
  - Skills: [`tests`] - Reason: extend dialog tests alongside the refactor
  - Omitted: [`playwright`] - Reason: existing component tests can fully validate this behavior

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: none | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - State setup: `client/src/components/Dialogs/SettingsDialog.tsx:114-120`
  - Concern derivation: `client/src/components/Dialogs/SettingsDialog.tsx:226-239`
  - Problematic effects: `client/src/components/Dialogs/SettingsDialog.tsx:241-260`
  - Toggle handlers: `client/src/components/Dialogs/SettingsDialog.tsx:353-379`
  - Test pattern: `client/src/components/Dialogs/SettingsDialog.test.tsx:93-130`

  **Acceptance Criteria** (agent-executable only):
  - [ ] `npx eslint client/src/components/Dialogs/SettingsDialog.tsx client/src/components/Dialogs/SettingsDialog.test.tsx` exits `0`
  - [ ] `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx"` exits `0` from `client/`
  - [ ] The file no longer triggers `react-hooks/set-state-in-effect` at former lines `244` and `256`

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Search-driven auto-expansion still works after the refactor
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx" --testNamePattern="filters concerns and sections from the search box"`
    Expected: Exit code 0 and the existing search test passes with the derived expanded concern set
    Evidence: .sisyphus/evidence/task-3-settings-search.txt

  Scenario: Legacy tab mapping still resolves and persists the correct concern
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx" --testNamePattern="maps legacy tabs into concern groups and persists the resolved concern"`
    Expected: Exit code 0 and `settingsActiveTab` still persists `governance` for the legacy `administration` tab
    Evidence: .sisyphus/evidence/task-3-settings-persistence.txt
  ```

  **Commit**: YES | Message: `fix(settings-dialog): derive expanded concern state` | Files: `[client/src/components/Dialogs/SettingsDialog.tsx, client/src/components/Dialogs/SettingsDialog.test.tsx]`

- [x] 4. Derive active section selection in `SettingsDialog` instead of setting it inside the observer effect

  **What to do**: Rename the mutable active-section state to an explicit requested/observed id, then derive the effective active section from the current concern’s section ids. Remove the effect-side `setActiveSectionId(sectionIds[0])` at lines `319-321`; instead, compute the fallback first section when the current stored id is absent/invalid. Keep the `IntersectionObserver` callback and `jumpToSection` updating the mutable requested id, but update render-time lookups and highlight logic to use the derived effective id.
  **Must NOT do**: Do not weaken the observer logic, do not remove programmatic scroll protection, and do not add lint suppressions.

  **Recommended Agent Profile**:
  - Category: `unspecified-low` - Reason: medium-complexity state refactor in one file with existing test scaffolding
  - Skills: [`tests`] - Reason: requires extending SettingsDialog coverage for section fallback/highlight behavior
  - Omitted: [`playwright`] - Reason: targeted component tests are sufficient for this refactor

  **Parallelization**: Can Parallel: NO | Wave 1 | Blocks: none | Blocked By: [3]

  **References** (executor has NO interview context - be exhaustive):
  - Observer effect: `client/src/components/Dialogs/SettingsDialog.tsx:275-325`
  - Section jump handler: `client/src/components/Dialogs/SettingsDialog.tsx:381-390`
  - Active section label/render usage: `client/src/components/Dialogs/SettingsDialog.tsx:401-405`, `client/src/components/Dialogs/SettingsDialog.tsx:519`, `client/src/components/Dialogs/SettingsDialog.tsx:631`
  - Test harness: `client/src/components/Dialogs/SettingsDialog.test.tsx:42-91`

  **Acceptance Criteria** (agent-executable only):
  - [ ] `npx eslint client/src/components/Dialogs/SettingsDialog.tsx client/src/components/Dialogs/SettingsDialog.test.tsx` exits `0`
  - [ ] `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx"` exits `0` from `client/`
  - [ ] The file no longer triggers `react-hooks/set-state-in-effect` at former line `321`

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Initial active section falls back to the first visible section without effect-set state
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx" --testNamePattern="active section"`
    Expected: Exit code 0 and the new/updated test proves the first section becomes active when no valid persisted section exists
    Evidence: .sisyphus/evidence/task-4-settings-active-section.txt

  Scenario: Existing organization trigger guard still holds after section-state refactor
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx" --testNamePattern="does not invoke the organization delete trigger when the concern mounts"`
    Expected: Exit code 0; no regression in effect sequencing or concern mounting side effects
    Evidence: .sisyphus/evidence/task-4-settings-delete-trigger.txt
  ```

  **Commit**: YES | Message: `fix(settings-dialog): derive active section state` | Files: `[client/src/components/Dialogs/SettingsDialog.tsx, client/src/components/Dialogs/SettingsDialog.test.tsx]`

- [x] 5. Replace impure sidebar skeleton randomness with deterministic width generation

  **What to do**: In `client/src/components/ui/sidebar.tsx`, replace the render-time `Math.random()` usage in `SidebarMenuSkeleton` with a deterministic width generator seeded from stable component data (default: `React.useId()` + a small local hash helper that maps to `50-89%`). Keep the visual variation, but make it idempotent across re-renders. Add `client/src/components/ui/sidebar.test.tsx` to assert the skeleton width CSS variable remains in-range and stable across rerenders.
  **Must NOT do**: Do not hard-disable the purity rule, do not remove the width variation entirely, and do not alter the exported component API.

  **Recommended Agent Profile**:
  - Category: `unspecified-low` - Reason: small behavioral refactor plus a new focused unit test
  - Skills: [`tests`] - Reason: new unit test is part of the task’s acceptance contract
  - Omitted: [`playwright`] - Reason: deterministic rendering can be validated in Vitest/JSDOM

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: none | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Problem area: `client/src/components/ui/sidebar.tsx:602-637`
  - Export surface: `client/src/components/ui/sidebar.tsx:701-726`
  - Existing test style: `client/src/components/Sidebar/sidebarUi.test.tsx:1-15`

  **Acceptance Criteria** (agent-executable only):
  - [ ] `npx eslint client/src/components/ui/sidebar.tsx client/src/components/ui/sidebar.test.tsx` exits `0`
  - [ ] `npx vitest run "src/components/ui/sidebar.test.tsx"` exits `0` from `client/`
  - [ ] `client/src/components/ui/sidebar.tsx` no longer triggers `react-hooks/purity` at former line `611`

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Sidebar skeleton width stays deterministic across rerenders
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/components/ui/sidebar.test.tsx"`
    Expected: Exit code 0; the test proves the rendered `--skeleton-width` value is stable on rerender and remains within the expected 50-89% range
    Evidence: .sisyphus/evidence/task-5-sidebar-skeleton.txt

  Scenario: Full lint no longer reports impure render-time randomness
    Tool: Bash
    Steps: Run `npx eslint client/src/components/ui/sidebar.tsx`
    Expected: Exit code 0; no `react-hooks/purity` violation remains for `SidebarMenuSkeleton`
    Evidence: .sisyphus/evidence/task-5-sidebar-lint.txt
  ```

  **Commit**: YES | Message: `fix(sidebar): make skeleton width deterministic` | Files: `[client/src/components/ui/sidebar.tsx, client/src/components/ui/sidebar.test.tsx]`

- [x] 6. Render login domain and certificate expiry in the shared public secret view

  **What to do**: First verify whether `client/src/components/secrets/SecretPayloadView.tsx` exists on the working branch. If it does not exist, create `client/src/components/secrets/SecretPayloadView.tsx` (and the `client/src/components/secrets/` directory if needed) so it satisfies the existing import in `client/src/pages/PublicSharePage.tsx`. If `.gitignore` still blocks that path, add or retain the repo-specific negation needed so `git check-ignore -v client/src/components/secrets/SecretPayloadView.tsx` returns no match. Then implement the shared renderer so the `LOGIN` branch renders `domain` between `URL` and `Notes` only when present, and the `CERTIFICATE` branch renders payload-level `expiresAt` as a formatted date string. Extend `client/src/pages/PublicSharePage.test.tsx` with focused cases that prove both values appear for external shares.
  **Must NOT do**: Do not change the order or masking behavior of existing fields, and do not conflate payload-level `CertificateData.expiresAt` with top-level secret expiry from list/detail metadata.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: small renderer augmentation with direct existing patterns to follow
  - Skills: [`tests`] - Reason: update page tests in the same task
  - Omitted: [`playwright`] - Reason: renderer behavior is fully covered by the page-level Vitest tests here

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: none | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Source-of-truth types: `client/src/api/secrets.api.ts:6-13`, `client/src/api/secrets.api.ts:25-33`
  - Existing detail renderer pattern: `client/src/components/Keychain/SecretDetailView.tsx:141-170`
  - Existing editor payload shape: `client/src/components/Keychain/SecretDialog.tsx:143-164`, `client/src/components/Keychain/SecretDialog.tsx:184-192`
  - Consumer page and missing import target: `client/src/pages/PublicSharePage.tsx:6`, `client/src/pages/PublicSharePage.tsx:101-110`
  - Test file to extend: `client/src/pages/PublicSharePage.test.tsx:31-76`

  **Acceptance Criteria** (agent-executable only):
  - [ ] `npx vitest run "src/pages/PublicSharePage.test.tsx"` exits `0` from `client/`
  - [ ] `npm run typecheck -w client` exits `0` with `PublicSharePage.tsx` successfully resolving `@/components/secrets/SecretPayloadView`
  - [ ] The public share renderer displays `Domain` for `LOGIN` payloads when `data.domain` is present
  - [ ] The public share renderer displays formatted certificate expiry when `data.expiresAt` is present

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Public share login payload shows domain when provided
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/pages/PublicSharePage.test.tsx" --testNamePattern="login domain"`
    Expected: Exit code 0; the new/updated test asserts `Username`, `Password`, and `Domain` render for a `LOGIN` payload with `domain`
    Evidence: .sisyphus/evidence/task-6-public-share-login-domain.txt

  Scenario: Public share certificate payload formats expiry cleanly
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/pages/PublicSharePage.test.tsx" --testNamePattern="certificate expiry"`
    Expected: Exit code 0; the new/updated test asserts the certificate branch renders expiry text derived from `data.expiresAt`
    Evidence: .sisyphus/evidence/task-6-public-share-cert-expiry.txt
  ```

  **Commit**: YES | Message: `fix(public-share): render domain and certificate expiry` | Files: `[client/src/components/secrets/SecretPayloadView.tsx, client/src/pages/PublicSharePage.test.tsx]`

- [x] 7. Render API key headers in the shared public secret view and keep secure-note behavior unchanged

  **What to do**: Continue from the created or existing `client/src/components/secrets/SecretPayloadView.tsx` in Task 6. Update the `API_KEY` branch so it renders `headers` when present, using a deterministic string form compatible with the current `ValueField` API (default: `JSON.stringify(headers, null, 2)` as a multiline value). Extend `client/src/pages/PublicSharePage.test.tsx` to assert the headers render. Leave `SECURE_NOTE` plain-text behavior unchanged, but make that intentional by verifying it through test coverage rather than changing sensitivity semantics.
  **Must NOT do**: Do not mark secure notes as sensitive in this PR, do not mask API key headers, and do not change existing `API Key`, `Endpoint`, or `Notes` labels.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: narrow renderer change with page-level test updates
  - Skills: [`tests`] - Reason: extend shared page tests in lockstep with the renderer update
  - Omitted: [`playwright`] - Reason: no browser automation needed for this renderer-level branch coverage

  **Parallelization**: Can Parallel: NO | Wave 2 | Blocks: none | Blocked By: [6]

  **References** (executor has NO interview context - be exhaustive):
  - Source-of-truth type: `client/src/api/secrets.api.ts:35-40`, `client/src/api/secrets.api.ts:43-46`
  - Existing detail renderer pattern: `client/src/components/Keychain/SecretDetailView.tsx:173-192`
  - Extension parity reference: `extra-clients/browser-extensions/src/popup/components/SecretDetail.tsx:160-176`
  - Consumer page: `client/src/pages/PublicSharePage.tsx:101-110`
  - Test file to extend: `client/src/pages/PublicSharePage.test.tsx:31-76`

  **Acceptance Criteria** (agent-executable only):
  - [ ] `npx vitest run "src/pages/PublicSharePage.test.tsx"` exits `0` from `client/`
  - [ ] The public share renderer displays headers for `API_KEY` payloads when `data.headers` is present
  - [ ] Existing `SECURE_NOTE` rendering remains plain `Content` output and no new masking toggle is introduced

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Public share API key payload shows headers in a readable multiline form
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/pages/PublicSharePage.test.tsx" --testNamePattern="api key headers"`
    Expected: Exit code 0; the new/updated test asserts the `Headers` label and serialized header content render for an `API_KEY` payload
    Evidence: .sisyphus/evidence/task-7-public-share-api-headers.txt

  Scenario: Secure note behavior remains intentionally unchanged
    Tool: Bash
    Steps: From `client/`, run `npx vitest run "src/pages/PublicSharePage.test.tsx" --testNamePattern="secure note content"`
    Expected: Exit code 0; the new/updated test proves `SECURE_NOTE` still renders `Content` in plain form without a reveal toggle
    Evidence: .sisyphus/evidence/task-7-public-share-secure-note.txt
  ```

  **Commit**: YES | Message: `fix(public-share): render api key headers` | Files: `[client/src/components/secrets/SecretPayloadView.tsx, client/src/pages/PublicSharePage.test.tsx]`

- [x] 8. Align `docker-build.yml` publish behavior with the documented branch/tag policy

  **What to do**: In `.github/workflows/docker-build.yml`, keep the PR trigger reduction already introduced by PR #602, but make the publish-profile logic decision-complete: `develop` sets `publish_latest=true`; `main` sets `publish_stable=true` and does **not** enable branch-ref publishing; `v*` tags publish semver tags only when `git branch -r --contains "$GITHUB_SHA"` confirms ancestry on `origin/main`. Preserve `fetch-depth: 0`, reusable verify/security jobs, and PR metadata tags.
  **Must NOT do**: Do not reintroduce `staging`, do not remove `fetch-depth: 0`, do not change verify/security job commands, and do not restore `{{is_default_branch}}` logic.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: bounded YAML/config fix in one workflow file
  - Skills: [] - Reason: no extra skill is required beyond careful config editing and command verification
  - Omitted: [`tests`] - Reason: verification is command-driven; no test file additions required

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: none | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Trigger and path filters: `.github/workflows/docker-build.yml:3-49`
  - Reusable verify gate usage: `.github/workflows/docker-build.yml:85-90`
  - Build/publish job region: `.github/workflows/docker-build.yml:112-264`
  - Shared verify contract: `.github/workflows/verify.yml:3-55`
  - Policy source: PR #602 body states `develop -> latest`, `main -> stable`, `v* on origin/main -> semver`

  **Acceptance Criteria** (agent-executable only):
  - [ ] A static assertion script over `.github/workflows/docker-build.yml` confirms: push/pull_request branches are `develop` and `main`; `fetch-depth: 0` remains; `type=raw,value=latest` and `type=raw,value=stable` remain; and the `main` branch case no longer enables branch-ref publishing
  - [ ] The file still contains the `git branch -r --contains "${GITHUB_SHA}"` ancestry gate for semver publishing

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Docker workflow encodes the correct publish profile rules
    Tool: Bash
    Steps: Run a Python assertion script that reads `.github/workflows/docker-build.yml` and checks for `branches: [develop, main]`, `fetch-depth: 0`, `publish_latest=true`, `publish_stable=true`, semver enable flags, and absence of branch-tag enablement in the `refs/heads/main` case
    Expected: Exit code 0; the workflow text matches the target policy exactly
    Evidence: .sisyphus/evidence/task-8-docker-workflow-policy.txt

  Scenario: Docker workflow still guards semver publishes behind origin/main ancestry
    Tool: Bash
    Steps: Run a Python assertion script that checks `.github/workflows/docker-build.yml` still contains the `git branch -r --contains "${GITHUB_SHA}"` gate inside the tag branch of the publish-profile step
    Expected: Exit code 0; semver publishing cannot happen for tags that are not on `origin/main`
    Evidence: .sisyphus/evidence/task-8-docker-workflow-semver-gate.txt
  ```

  **Commit**: YES | Message: `ci(docker-build): publish stable only from main` | Files: `[.github/workflows/docker-build.yml]`

- [x] 9. Align `gateways-build.yml` publish behavior with the documented branch/tag policy

  **What to do**: Mirror the same policy decisions in `.github/workflows/gateways-build.yml`: `develop` emits `latest`, `main` emits `stable` only, semver tags require ancestry on `origin/main`, and `fetch-depth: 0` stays intact for the ancestry check. Preserve the gateway Go verify loop and the existing build/scan structure.
  **Must NOT do**: Do not modify gateway verify commands, do not add/remove gateway matrix entries, and do not diverge from the Docker workflow’s policy naming.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: targeted config mirror of Task 8 in a second workflow
  - Skills: [] - Reason: no additional skill injection is needed for this YAML-only task
  - Omitted: [`tests`] - Reason: command assertions are sufficient here

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: none | Blocked By: none

  **References** (executor has NO interview context - be exhaustive):
  - Trigger filters: `.github/workflows/gateways-build.yml:3-15`
  - Gateway verify loop: `.github/workflows/gateways-build.yml:22-52`
  - Build/publish job region: `.github/workflows/gateways-build.yml:53-154`
  - Policy mirror reference: `.github/workflows/docker-build.yml:112-264`

  **Acceptance Criteria** (agent-executable only):
  - [ ] A static assertion script over `.github/workflows/gateways-build.yml` confirms: push/pull_request branches are `develop` and `main`; `fetch-depth: 0` remains; `latest` and `stable` tags remain; and the `main` branch case no longer enables branch-ref publishing
  - [ ] The file still contains the `git branch -r --contains "${GITHUB_SHA}"` ancestry gate for semver publishing

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Gateway workflow encodes the correct publish profile rules
    Tool: Bash
    Steps: Run a Python assertion script that reads `.github/workflows/gateways-build.yml` and checks for `branches: [develop, main]`, `fetch-depth: 0`, `publish_latest=true`, `publish_stable=true`, semver enable flags, and absence of branch-tag enablement in the `refs/heads/main` case
    Expected: Exit code 0; the workflow text matches the target policy exactly
    Evidence: .sisyphus/evidence/task-9-gateway-workflow-policy.txt

  Scenario: Gateway workflow still guards semver publishes behind origin/main ancestry
    Tool: Bash
    Steps: Run a Python assertion script that checks `.github/workflows/gateways-build.yml` still contains the `git branch -r --contains "${GITHUB_SHA}"` gate inside the tag branch of the publish-profile step
    Expected: Exit code 0; semver publishing cannot happen for tags that are not on `origin/main`
    Evidence: .sisyphus/evidence/task-9-gateway-workflow-semver-gate.txt
  ```

  **Commit**: YES | Message: `ci(gateways-build): publish stable only from main` | Files: `[.github/workflows/gateways-build.yml]`

## Final Verification Wave (MANDATORY — after ALL implementation tasks)
> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback -> fix -> re-run -> present again -> wait for okay.
- [x] F1. Plan Compliance Audit — oracle

  **What to do**: Run an oracle review against the completed diff plus `.sisyphus/plans/pr-602-fixes.md` and verify every touched file maps back to Tasks 1-9 with no unplanned scope expansion.
  **Acceptance Criteria**:
  - [ ] Oracle returns approval with zero critical scope mismatches
  - [ ] Every modified file is attributable to exactly one planned task or shared verification artifact

  **QA Scenarios**:
  ```
  Scenario: Planned scope audit passes
    Tool: Bash
    Steps: Capture `git diff --name-only "$(git merge-base HEAD develop)"...HEAD` and `git diff --stat "$(git merge-base HEAD develop)"...HEAD`, then feed the diff summary plus `.sisyphus/plans/pr-602-fixes.md` into the oracle compliance review
    Expected: Oracle confirms the implemented diff stays within Tasks 1-9 and flags no out-of-scope file edits
    Evidence: .sisyphus/evidence/f1-plan-compliance.txt

  Scenario: No planned task was skipped silently
    Tool: Bash
    Steps: Capture `npm run lint`, `npm run typecheck -w client`, `npx vitest run "src/components/Dialogs/SettingsDialog.test.tsx"`, `npx vitest run "src/pages/PublicSharePage.test.tsx"`, and `npx vitest run "src/components/ui/sidebar.test.tsx"`, then include the outputs in the oracle review packet
    Expected: Oracle confirms each task’s acceptance evidence exists and corresponds to the plan
    Evidence: .sisyphus/evidence/f1-acceptance-evidence.txt
  ```

- [x] F2. Code Quality Review — unspecified-high

  **What to do**: Run an independent code-quality review over the final diff, focusing on accidental regressions, unnecessary churn, brittle test additions, and whether the SettingsDialog/sidebar refactors preserved behavior cleanly.
  **Acceptance Criteria**:
  - [ ] Reviewer reports no critical maintainability or regression risks
  - [ ] Any minor follow-ups are documented but non-blocking

  **QA Scenarios**:
  ```
  Scenario: Final diff passes independent quality review
    Tool: Bash
    Steps: Capture `git diff --unified=3 "$(git merge-base HEAD develop)"...HEAD` and provide it to the code-quality review agent together with the plan file
    Expected: Reviewer approves the refactors and renderer/workflow changes with no critical findings
    Evidence: .sisyphus/evidence/f2-code-quality-review.txt

  Scenario: Verification artifacts support the quality review
    Tool: Bash
    Steps: Capture fresh outputs for `npm run lint` and `npm run verify`, attach them to the quality review packet
    Expected: Reviewer confirms the diff is validated by the expected repo-level checks
    Evidence: .sisyphus/evidence/f2-quality-checks.txt
  ```

- [x] F3. Real Manual QA — unspecified-high (+ playwright if UI)

  **What to do**: Execute real UI QA against the local stack for the shared public-secret flows and any touched settings/sidebar behavior that can be exercised safely, plus rerun the matching command-line verification suite.
  **Acceptance Criteria**:
  - [ ] UI QA confirms public share flows render the newly added fields correctly
  - [ ] Repo verification commands remain green after browser QA

  **QA Scenarios**:
  ```
  Scenario: Public share UI renders the fixed payload fields on a live page
    Tool: Playwright
    Steps: Start from a local app instance, navigate to a controlled public-share route or seeded test page that exercises `LOGIN.domain`, `CERTIFICATE.expiresAt`, and `API_KEY.headers`, then assert the page shows the `Domain`, formatted certificate expiry, and `Headers` labels/content without masking secure-note content
    Expected: All three payload branches render exactly as planned and no console/runtime error occurs
    Evidence: .sisyphus/evidence/f3-public-share-ui.png

  Scenario: Repo-level verification still passes after UI checks
    Tool: Bash
    Steps: Run `npm run verify` from repo root after the browser QA completes
    Expected: Exit code 0; no verification step regressed after the implemented fixes
    Evidence: .sisyphus/evidence/f3-repo-verify.txt
  ```

- [x] F4. Scope Fidelity Check — deep

  **What to do**: Run a deep scope audit to ensure the final implementation fixed only the approved problems: renderer gaps, blocking lint errors, SettingsDialog/sidebar purity errors, and Docker/Gateway publish policy.
  **Acceptance Criteria**:
  - [ ] Deep review confirms no warning-backlog cleanup or unrelated workflow redesign slipped in
  - [ ] Deep review confirms `SECURE_NOTE` semantics and `fetch-depth: 0` remained intentionally unchanged

  **QA Scenarios**:
  ```
  Scenario: Deep scope audit confirms no warning-only cleanup leaked into the patch
    Tool: Bash
    Steps: Capture `git diff --name-only "$(git merge-base HEAD develop)"...HEAD` and `git diff --unified=0 "$(git merge-base HEAD develop)"...HEAD`, then run the deep scope review against the plan file
    Expected: Reviewer confirms the patch stays limited to planned problem areas and warning-only files were not broadened unnecessarily
    Evidence: .sisyphus/evidence/f4-scope-audit.txt

  Scenario: Explicit non-changes remain intact
    Tool: Bash
    Steps: Capture the final contents of `.github/workflows/docker-build.yml`, `.github/workflows/gateways-build.yml`, and the shared secret renderer; include them in the deep review packet with the plan’s Must NOT Have list
    Expected: Reviewer confirms `SECURE_NOTE` stayed plain, `fetch-depth: 0` stayed present, and no workflow redesign beyond the documented policy fix occurred
    Evidence: .sisyphus/evidence/f4-non-change-audit.txt
  ```

## Commit Strategy
- Prefer one commit per completed task unless two consecutive tasks touch the same file and are intentionally landed together:
  - `lint(client): remove unused imports in dialogs and keychain`
  - `lint(client): remove remaining unused vars and imports`
  - `fix(settings-dialog): derive expanded concerns without effect setters`
  - `fix(settings-dialog): derive active section state without effect setters`
  - `fix(sidebar): make skeleton width deterministic`
  - `fix(public-share): render login domain and certificate expiry`
  - `fix(public-share): render api key headers in shared payload view`
  - `ci(docker-build): publish stable only from main`
  - `ci(gateways-build): publish stable only from main`

## Success Criteria
- All confirmed PR #602 review findings classified as either fixed or intentionally left unchanged with rationale.
- Repo lint exits cleanly with zero errors.
- Shared public secret rendering matches the required subset of `SecretPayload` fields already supported by the editor/detail views.
- Workflow behavior matches documented policy: `develop -> latest`, `main -> stable`, `v* on origin/main -> semver`.
- No acceptance criterion requires manual judgment to determine pass/fail.
