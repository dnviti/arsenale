# Draft: Fix PR #602 Issues

## Requirements (confirmed)
- Fix all issues identified in PR #602 review so CI passes and review comments are addressed

## Issues Identified

### Pre-existing CI Blockers (not introduced by PR)
1. **sidebar.tsx:611** — `Math.random()` inside `useMemo` violates `react-hooks/purity`
   - Component: `SidebarMenuSkeleton` (shadcn/ui sidebar, line 602-638)
   - Used for skeleton loading widths (random between 50-90%)
   - Fix options: deterministic width array, seeded approach, or CSS-only
2. **LoginPage.test.tsx:287** — `view` assigned but never used (`@typescript-eslint/no-unused-vars`)
   - Test: "remembers passkey preference after successful passkey login"
   - `const view = renderLoginPage()` — result not used after render
   - Fix: prefix with `_` → `const _view = renderLoginPage()`

### PR-specific Issues (from review comments)
3. **SecretPayloadView.tsx** — Missing fields from SecretPayload types:
   - LOGIN: missing `domain` (optional in `LoginData`)
   - CERTIFICATE: missing `expiresAt` (optional string in `CertificateData`)
   - API_KEY: missing `headers` (optional `Record<string, string>` in `ApiKeyData`)
   - SECURE_NOTE: `content` not marked `sensitive`
   - Used only in: `PublicSharePage.tsx` (line 109)
4. **docker-build.yml & gateways-build.yml** — Tagging strategy flaw:
   - `publish_branch=true` on main → `type=ref,event=branch` emits `:main` alongside `:stable`
   - Deployment default: `arsenale_image_tag: latest` (from `inventory/group_vars/all/vars.yml:66`)
   - Fix: remove `publish_branch` flag entirely; main should only get `:stable`
5. **fetch-depth: 0** — Full clone on every matrix build (low priority, perf only)

## Technical Decisions
- sidebar.tsx: TBD — deterministic array vs CSS approach
- LoginPage.test.tsx: prefix with `_` (standard pattern)
- SecretPayloadView: add missing ValueField renders + mark SECURE_NOTE as sensitive
- CI workflows: remove `publish_branch`, disable `type=ref,event=branch` for main
- fetch-depth: conditional deep fetch only on v* tags

## Research Findings
- `SidebarMenuSkeleton` is shadcn/ui's standard component — upstream has same pattern
- `arsenale_image_tag: latest` is the deployment default — `:latest` from develop is critical
- `SecretPayloadView` has single consumer: `PublicSharePage.tsx`
- Deployment compose uses `arsenale_registry/service:arsenale_image_tag` pattern

## Open Questions
- Should pre-existing lint errors be fixed in the same PR #602 or a separate commit?
- sidebar.tsx: deterministic widths or CSS-only approach?
- SECURE_NOTE: should `content` be `sensitive`? (design choice)

## Scope Boundaries
- INCLUDE: All 5 issues above
- EXCLUDE: Other pre-existing lint warnings (96 warnings are not blocking)
