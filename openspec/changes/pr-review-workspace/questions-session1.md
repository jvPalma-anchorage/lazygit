# Refinement Questions — Session 1

| Field     | Value |
|-----------|-------|
| Session   | 1 |
| Date      | 2026-06-29 |
| Change ID | pr-review-workspace |

## Analysis Summary

Agents ran: 1 Spec-Completeness, 2 Problem-&-Context, 3 Requirements-Coverage, 4 System-&-Infrastructure, 5 Goal-&-Behaviour, 6 Documentation-Completeness (all 6, in parallel, read-only).
Total findings: ~45 across the six agents.
Breakdown: 9 critical, ~22 important, ~14 minor.

**Self-resolving without you** (will be applied on the next `/opsx:refine` cycle — no decision needed): add the missing v1 line-comment-surface requirement to the spec; mark deferred requirements; add a repo-identity-verification requirement; formalize gh-dash-filter injection-safety (argv, never shell); document the pr-review-mode→workspace supersession; port the data-shape docs (PR metadata / threads / check-runs / diff-patch / fetch command / window-context wiring / viewed-state struct) from `pr-review-mode/docs/`; fix two stale line-numbers (`AddReviewComment` → `github_review.go:105`; `WindowName` → `pr_review_context.go:114`); break task 1.x into explicit threading sub-items; specify initial focus + per-context keybindings from your original brief (tree `space`=viewed, `enter`=open file / collapse folder; diff `space`/`shift+↑↓`=range-select, `c`=comment).

The 7 questions below are the genuine decisions. They drive a lot of downstream scenarios, so answering them unblocks the rest.

## Instructions

Write your answer on the blank line after each **Answer:** marker. For option questions, write a letter (e.g. `A`) or a custom answer. Write `SKIP` to defer a topic. When done, re-run `/opsx:refine pr-review-workspace` to apply your decisions.

---

## Questions

### Q1 [CRITICAL] — Diff source: fetch PR refs locally, or render the GitHub API patch?

This is the load-bearing decision (design DW2). To make review "mirror the usual stage/unstage logic," the plan **fetches the PR head+base into `refs/lazygit-review/<pr>/*` inside your checkout**, then renders diffs via real `git diff` through delta and reuses git for the PR-commits list and line-selection. This **mutates your repo** (writes refs, runs `git fetch` — never touches HEAD/index/branches). The alternative renders the GitHub API patch text through delta via a PTY: zero repo mutation, but no true `git diff` fidelity, no PR-commits drill-in, and line→GitHub mapping stays bespoke.

**Affected files**: `design.md` (DW2), `tasks.md` (phases 2,3,5), `specs/…/spec.md`

**Options**:
- A) **Local-ref fetch (recommended)** — true delta diffs + PR commits + staging-like selection; isolated refs, pruned on boot; repo mutated (refs only).
- B) **API-patch through pager** — pristine repo, no fetch; loses the PR-commits tab and git-native selection; bespoke line mapping.
- C) **Hybrid** — ship API-patch preview for v1, add the local-ref fetch later when the PR-commits phase lands.

**Answer:**


---

### Q2 [IMPORTANT] — Relationship to your existing gh-dash `L` worktree flow

Your gh-dash config has **`L`** (fetch the PR into `/home/user/anc-review`, soft-reset to merge-base, launch lazygit) and **`R`** (`llg-dev` review mode). If Q1=A, review mode's fetch overlaps with what `L` already does.

**Affected files**: `design.md` (Open Questions), `docs/gh-dash-integration` (to be ported)

**Options**:
- A) **`R` replaces `L`** — retire the `L` worktree dance; review mode is the one flow.
- B) **Coexist** — keep both; review mode uses its own `refs/lazygit-review/*` and never touches the `anc-review` worktree.
- C) **Reuse the `anc-review` worktree** — review mode operates in that worktree instead of fetching into the launch checkout.

**Answer:**


---

### Q3 [IMPORTANT] — Cleanup / retention policy for review refs and viewed-state

`refs/lazygit-review/*` and the persisted viewed-state both accumulate per PR over time (Agents 3 & 5).

**Affected files**: `tasks.md` (2.4, 4.3), `specs/…/spec.md`

**Options**:
- A) **Prune on boot (recommended)** — drop review refs older than ~30 days and viewed-state for merged/closed PRs, on startup.
- B) **Eager cleanup** — delete a PR's refs on quit / PR-switch; keep viewed-state until the PR is merged/closed.
- C) **Leave everything** — cheapest, unbounded growth, revisit if it bites.

**Answer:**


---

### Q4 [IMPORTANT] — Viewed-state identity (and binary / deleted / renamed files)

GitHub's "viewed" checkbox clears when a file changes. The key must handle modified/added/deleted/renamed/binary files (Agents 1, 3, 4, 6).

**Affected files**: `design.md` (DW4), `tasks.md` (4.1–4.2), `docs/viewed-state-schema` (to be written)

**Options**:
- A) **Blob OID (recommended)** — one `git rev-parse`, matches GitHub; deleted files keyed by `previousPath+status` (not re-viewable once gone); a rename = new path, mark clears.
- B) **Patch hash** — also covers binaries, but recomputed every load.
- C) **Hybrid** — blob OID for normal files, patch-hash fallback for binary/large files.

**Answer:**


---

### Q5 [IMPORTANT] — Behavior when `gh` / auth / network fails

Every panel makes a `gh` call, and this feature already hit an infinite "Loading…" once (design risk). The spec currently has zero failure scenarios.

**Affected files**: `specs/…/spec.md` (new error scenarios), `tasks.md`

**Options**:
- A) **Fail-fast at boot, toast later (recommended)** — if `gh` is missing/unauthenticated or the initial load fails, exit with an actionable message; once loaded, per-panel errors show a toast + retry.
- B) **Never block boot** — every panel independently shows Loading → error/empty with a retry, including the first one.
- C) **Auth-only fail-fast** — block boot only on auth failure; treat later errors as silent empty states.

**Answer:**


---

### Q6 [IMPORTANT] — v1 scope, and how deferred tabs appear in the spec

The spec lists Conversation / Checks / grouped-review as full requirements, but tasks defer them to phases 6–8 (Agents 1, 3, 5 flagged the mismatch). Also, v1's PR-list `[1]` shows only the launched PR; the multi-section gh-dash browser is phase 8.

**Affected files**: `specs/…/spec.md`, `tasks.md`

**Options**:
- A) **Confirm v1 = phases 1–5** (layout · local-ref preview · line-comment surface + standalone posting · viewed-state · PR commits) and annotate Conversation/Checks/grouped-review/multi-section-list as `[DEFERRED — Phase N]` in this same spec (recommended).
- B) **Split deferred requirements into a separate v2 spec file**; this change's spec covers only v1.
- C) **Promote one of {Conversation, Checks, Commits} into v1** — say which, if it's higher priority than assumed.

**Answer:**


---

### Q7 [MINOR] — Status window and large-PR guardrails in review mode

`refresh_helper` always refreshes the Status window; a huge PR (thousands of files / massive diff) could hang the preview (Agents 3, 5).

**Affected files**: `tasks.md` (1.7), `specs/…/spec.md`

**Options**:
- A) **Suppress Status + cap (recommended)** — hide the Status window in review mode; cap the files tree / diff preview above a threshold with a "truncated, N more" note.
- B) **Minimal status line, no caps** — keep a small branch/identity line; accept slowness on huge PRs in v1.
- C) **Suppress Status, no caps** — revisit caps later if it bites.

**Answer:**


---

## Processing Log (filled automatically on re-run)

- [x] All answers parsed (all 7 = A; project-agnostic + 10/2000 cap notes captured)
- [x] Specs updated
- [x] Design decisions recorded (D1.1–D1.7)
- [x] Tasks updated
- [x] Docs updated (REFERENCES, data-shapes, review-fetch-command, viewed-state-schema)
- [x] Refinement log appended
